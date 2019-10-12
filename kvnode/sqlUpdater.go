package kvnode

import (
	"database/sql/driver"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/sniperHW/flyfish/proto"
	"github.com/sniperHW/flyfish/util/fixedarray"
	"github.com/sniperHW/flyfish/util/str"
	"github.com/sniperHW/kendynet/util"
	"net"
	"sync/atomic"
	"time"
)

var (
	errServerStop = fmt.Errorf("errServerStop")
	errLoseLease  = fmt.Errorf("errLoseLease")
)

type updatePending struct {
	sqlStr *str.Str
	kvs    *fixedarray.FixedArray
	rn     *raftNode
}

type sqlUpdater struct {
	db        *sqlx.DB
	name      string
	lastTime  time.Time
	queue     *util.BlockQueue
	sqlMgr    *sqlMgr
	localList []interface{}
	pending   updatePending
}

func newSqlUpdater(sqlMgr *sqlMgr, db *sqlx.DB, name string) *sqlUpdater {
	sqlMgr.sqlUpdateWg.Add(1)
	return &sqlUpdater{
		name:      name,
		queue:     util.NewBlockQueueWithName(name),
		db:        db,
		sqlMgr:    sqlMgr,
		localList: []interface{}{},
		pending: updatePending{
			sqlStr: str.NewStr(make([]byte, 1024*1024), 0),
			kvs:    fixedarray.NewFixedArray(200),
		},
	}
}

func isRetryError(err error) bool {
	if err == driver.ErrBadConn {
		return true
	} else {
		switch err.(type) {
		case *net.OpError:
			return true
		case net.Error:
			return true
		default:
		}
	}
	return false
}

func (this *sqlUpdater) reset() {
	this.pending.sqlStr.Reset()
	this.pending.kvs.Reset()
	this.pending.rn = nil
}

func (this *sqlUpdater) run() {
	for {
		closed, localList := this.queue.Get()

		for _, v := range localList {
			this.append(v)
		}

		if !this.pending.kvs.Empty() {
			this.exec()
		}

		for {
			if len(this.localList) > 0 {
				localList = this.localList
				this.localList = []interface{}{}
				for _, v := range localList {
					this.append(v)
				}

				if !this.pending.kvs.Empty() {
					this.exec()
				}

			} else {
				break
			}

			if this.queue.Len() > 0 || len(this.localList) == 0 {
				break
			}
		}

		if closed {
			Infoln(this.name, "stoped")
			this.sqlMgr.sqlUpdateWg.Done()
			return
		}
	}
}

func (this *sqlUpdater) append(v interface{}) {
	switch v.(type) {
	case sqlPing:
		if time.Now().Sub(this.lastTime) > time.Second*5*60 {
			//空闲超过5分钟发送ping
			err := this.db.Ping()
			if nil != err {
				Errorln("ping error", err)
			}
			this.lastTime = time.Now()
		}
	case *kv:
		kv := v.(*kv)
		rn := kv.slot.getRaftNode()

		if !rn.hasLease() {
			kv.Lock()
			kv.setWriteBack(false)
			kv.Unlock()
			return
		}

		if this.pending.rn != nil && this.pending.rn != rn {
			this.exec()
		}

		this.pending.rn = rn
		this.pending.kvs.Append(kv)

		kv.Lock()

		tt := kv.getSqlFlag()
		if tt == sql_insert_update {
			this.sqlMgr.buildInsertUpdateString(this.pending.sqlStr, kv)
		} else if tt == sql_update {
			this.sqlMgr.buildUpdateString(this.pending.sqlStr, kv)
		} else if tt == sql_delete {
			this.sqlMgr.buildDeleteString(this.pending.sqlStr, kv)
		}

		kv.setSqlFlag(sql_none)

		if len(kv.modifyFields) > 0 {
			kv.modifyFields = map[string]*proto.Field{}
		}

		kv.Unlock()
	}

	if this.pending.kvs.Full() {
		this.exec()
	}
}

func (this *sqlUpdater) onSqlResult(kv *kv, err error) {
	if err == errServerStop {
		return
	} else {
		kv.Lock()
		defer kv.Unlock()
		if err == errLoseLease {
			kv.setWriteBack(false)
		} else {
			if sql_none == kv.getSqlFlag() {
				kv.setWriteBack(false)
			} else {
				//执行exec时再次发生变更
				this.localList = append(this.localList, kv)
			}
		}
	}
}

func (this *sqlUpdater) exec() {

	defer this.reset()

	rn := this.pending.rn

	if !rn.hasLease() {
		this.pending.kvs.ForEach(func(v interface{}) {
			kv := v.(*kv)
			kv.Lock()
			kv.setWriteBack(false)
			kv.Unlock()
		})
		return
	}

	var err error
	str := this.pending.sqlStr.ToString()

	atomic.AddInt64(&this.sqlMgr.totalUpdateSqlCount, int64(this.pending.kvs.Len()))

	for {
		_, err = this.db.Exec(str)
		if nil == err {
			break
		} else {
			Errorln(str, err)
			if isRetryError(err) {
				Errorln("sqlUpdater exec error:", err)
				if this.sqlMgr.isStoped() {
					err = errServerStop
					break
				}

				if !rn.hasLease() {
					//已经失去租约，不能再执行
					err = errLoseLease
					break
				}

				//休眠一秒重试
				time.Sleep(time.Second)
			} else {
				Errorln("sqlUpdater exec error:", err)
				break
			}
		}
	}

	Debugln("onSqlResult", err)

	this.pending.kvs.ForEach(func(v interface{}) {
		kv := v.(*kv)
		this.onSqlResult(kv, err)
	})
}
