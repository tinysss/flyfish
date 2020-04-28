package kvnode

//go test -covermode=count -v -coverprofile=coverage.out -run=.
//go tool cover -html=coverage.out

import (
	"fmt"
	"github.com/sniperHW/flyfish/client"
	"github.com/sniperHW/flyfish/conf"
	"github.com/sniperHW/flyfish/errcode"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
	"time"
)

//fixed the var below first
var _sqltype string = "pgsal" //or mysql
var _host string = "localhost"
var _port int = 5432
var _user string = "sniper"
var _password string = "802802"
var _db string = "test"

var configStr string = `
CacheGroupSize          = 1                  #cache分组数量，每一个cache组单独管理，以降低处理冲突

MaxCachePerGroupSize    = 10               #每组最大key数量，超过数量将会触发key剔除

SqlLoadPipeLineSize     = 200                  #sql加载管道线大小

SqlLoadQueueSize        = 10000                #sql加载请求队列大小，此队列每CacheGroup一个

SqlLoaderCount          = 5
SqlUpdaterCount         = 5

StrInitCap              = 1048576 # 1mb

ServiceHost             = "127.0.0.1"

ServicePort             = 10018

ReplyBusyOnQueueFull    = false                #当处理队列满时是否用busy响应，如果填false,直接丢弃请求，让客户端超时 

Compress                = true

BatchByteSize           = 148576
BatchCount              = 200 
ProposalFlushInterval   = 100
ReadFlushInterval       = 10                   	 


[DBConfig]
SqlType         = "%s"


DbHost          = "%s"
DbPort          = %d
DbUser			= "%s"
DbPassword      = "%s"
DbDataBase      = "%s"

ConfDbHost      = "%s"
ConfDbPort      = %d
ConfDbUser      = "%s"
ConfDbPassword  = "%s"
ConfDataBase    = "%s"


[Log]
MaxLogfileSize  = 104857600 # 100mb
LogDir          = "log1"
LogPrefix       = "flyfish"
LogLevel        = "debug"
EnableLogStdout = true	
`

func test(t *testing.T, c *client.Client) {
	{
		//del
		//set
		fields := map[string]interface{}{}
		fields["age"] = 12
		fields["name"] = "sniperHW"

		r1 := c.Set("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r1.ErrCode)

		r2 := c.Del("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r2 = c.Del("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_RECORD_NOTEXIST, r2.ErrCode)

		r1 = c.Set("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r1.ErrCode)
		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r2 = c.Del("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)

		r2 = c.Del("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_RECORD_NOTEXIST, r2.ErrCode)

	}

	{
		//set

		fields := map[string]interface{}{}
		fields["age"] = 12
		fields["name"] = "sniperHW"

		r1 := c.Set("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r1.ErrCode)

		r2 := c.GetAll("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)
		assert.Equal(t, "sniperHW", r2.Fields["name"].GetString())
		assert.Equal(t, int64(12), r2.Fields["age"].GetInt())

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r1 = c.Set("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r1.ErrCode)

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r1 = c.Set("users1", "sniperHW", fields, 100).Exec()
		assert.Equal(t, errcode.ERR_VERSION_MISMATCH, r1.ErrCode)

	}

	{

		//CompareAndSetNx
		r1 := c.CompareAndSetNx("users1", "sniperHW", "age", 1, 10).Exec()
		assert.Equal(t, errcode.ERR_CAS_NOT_EQUAL, r1.ErrCode)

		r2 := c.CompareAndSetNx("users1", "sniperHW", "age", 12, 10).Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)

		r3 := c.GetAll("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r3.ErrCode)
		assert.Equal(t, int64(10), r3.Fields["age"].GetInt())

		r4 := c.Del("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r4.ErrCode)

		r5 := c.CompareAndSetNx("users1", "sniperHW", "age", 1, 100).Exec()
		assert.Equal(t, errcode.ERR_OK, r5.ErrCode)

		r6 := c.GetAll("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r6.ErrCode)
		assert.Equal(t, int64(100), r6.Fields["age"].GetInt())

		c.Del("users1", "sniperHW").Exec()
		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r5 = c.CompareAndSetNx("users1", "sniperHW", "age", 1, 100).Exec()
		assert.Equal(t, errcode.ERR_OK, r5.ErrCode)

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r5 = c.CompareAndSetNx("users1", "sniperHW", "age", 1, 100, 100).Exec()
		assert.Equal(t, errcode.ERR_VERSION_MISMATCH, r5.ErrCode)

		fields := map[string]interface{}{}
		fields["age"] = 12
		fields["name"] = "sniperHW"

		c.Set("users1", "sniperHW", fields).Exec()

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r5 = c.CompareAndSetNx("users1", "sniperHW", "age", 1, 100).Exec()
		assert.Equal(t, errcode.ERR_CAS_NOT_EQUAL, r5.ErrCode)

		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
		r5 = c.CompareAndSetNx("users1", "sniperHW", "age", 12, 10).Exec()
		assert.Equal(t, errcode.ERR_OK, r5.ErrCode)

	}

	{
		//setNx
		fields := map[string]interface{}{}
		fields["age"] = 12
		fields["name"] = "sniperHW"

		r1 := c.SetNx("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_RECORD_EXIST, r1.ErrCode)

		r2 := c.Del("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)

		r3 := c.SetNx("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r3.ErrCode)

		c.Del("users1", "sniperHW").Exec()
		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r3 = c.SetNx("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r3.ErrCode)

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r3 = c.SetNx("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_RECORD_EXIST, r3.ErrCode)

	}

	{
		//incr/decr
		r1 := c.GetAll("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r1.ErrCode)
		age := r1.Fields["age"].GetInt()

		r2 := c.IncrBy("users1", "sniperHW", "age", 1).Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)
		assert.Equal(t, age+1, r2.Fields["age"].GetInt())
		age = r2.Fields["age"].GetInt()

		r3 := c.DecrBy("users1", "sniperHW", "age", 1).Exec()
		assert.Equal(t, errcode.ERR_OK, r3.ErrCode)
		assert.Equal(t, age-1, r3.Fields["age"].GetInt())

		//del and kick
		c.Del("users1", "sniperHW").Exec()
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r2 = c.IncrBy("users1", "sniperHW", "age", 1).Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)

		//del and kick
		c.Del("users1", "sniperHW").Exec()
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r3 = c.DecrBy("users1", "sniperHW", "age", 1).Exec()
		assert.Equal(t, errcode.ERR_OK, r3.ErrCode)

		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r3 = c.DecrBy("users1", "sniperHW", "age", 1, 100).Exec()
		assert.Equal(t, errcode.ERR_VERSION_MISMATCH, r3.ErrCode)

		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r3 = c.DecrBy("users1", "sniperHW", "age", 1).Exec()
		assert.Equal(t, errcode.ERR_OK, r3.ErrCode)

	}

	{
		//version
		fields := map[string]interface{}{}
		fields["age"] = 12
		fields["name"] = "sniperHW"

		c.SetNx("users1", "sniperHW", fields).Exec()
		r1 := c.GetAll("users1", "sniperHW").Exec()
		assert.Equal(t, errcode.ERR_OK, r1.ErrCode)
		version := r1.Version

		r2 := c.Set("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)
		assert.Equal(t, version+1, r2.Version)

		r3 := c.GetAllWithVersion("users1", "sniperHW", version+1).Exec()
		assert.Equal(t, errcode.ERR_RECORD_UNCHANGE, r3.ErrCode)

		r4 := c.Set("users1", "sniperHW", fields, version).Exec()
		assert.Equal(t, errcode.ERR_VERSION_MISMATCH, r4.ErrCode)

	}

	{
		//reload
		for {
			r := c.ReloadTableConf().Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	{
		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
		//again
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	{
		//CompareAndSet
		fields := map[string]interface{}{}
		fields["age"] = 12
		fields["name"] = "sniperHW"

		r1 := c.Set("users1", "sniperHW", fields).Exec()
		assert.Equal(t, errcode.ERR_OK, r1.ErrCode)

		r2 := c.CompareAndSet("users1", "sniperHW", "age", 12, 10).Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)

		r3 := c.CompareAndSet("users1", "sniperHW", "age", 12, 10).Exec()
		assert.Equal(t, errcode.ERR_CAS_NOT_EQUAL, r3.ErrCode)
		assert.Equal(t, r3.Fields["age"].GetInt(), int64(10))

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r2 = c.CompareAndSet("users1", "sniperHW", "age", 10, 20).Exec()
		assert.Equal(t, errcode.ERR_OK, r2.ErrCode)

		r2 = c.CompareAndSet("users1", "sniperHW", "age", 20, 10, 1).Exec()
		assert.Equal(t, errcode.ERR_VERSION_MISMATCH, r2.ErrCode)

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r2 = c.CompareAndSet("users1", "sniperHW", "age", 12, 10, 1).Exec()
		assert.Equal(t, errcode.ERR_VERSION_MISMATCH, r2.ErrCode)

		//kick
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r2 = c.CompareAndSet("users1", "sniperHW", "age", 12, 10).Exec()
		assert.Equal(t, errcode.ERR_CAS_NOT_EQUAL, r2.ErrCode)

		//del and kick
		c.Del("users1", "sniperHW").Exec()
		for {
			r := c.Kick("users1", "sniperHW").Exec()
			if r.ErrCode == errcode.ERR_OK {
				break
			}
			time.Sleep(time.Millisecond * 100)
		}

		r2 = c.CompareAndSet("users1", "sniperHW", "age", 12, 10).Exec()
		assert.Equal(t, errcode.ERR_RECORD_NOTEXIST, r2.ErrCode)

	}

}

func TestKvnode2(t *testing.T) {
	//先删除所有kv文件
	os.RemoveAll("./kv-1-1")
	os.RemoveAll("./kv-1-1-snap")

	conf.LoadConfigStr(fmt.Sprintf(configStr, _sqltype, _host, _port, _user, _password, _db, _host, _port, _user, _password, _db))

	InitLogger()

	cluster := "http://127.0.0.1:12377"
	id := 1

	node := NewKvNode()

	if err := node.Start(&id, &cluster); nil != err {
		panic(err)
	}

	//等到所有store都成为leader之后再发送指令
	waitCondition(func() bool {
		node.storeMgr.RLock()
		defer node.storeMgr.RUnlock()
		for _, v := range node.storeMgr.stores {
			if !v.rn.isLeader() {
				return false
			}
		}
		return true
	})

	c := client.OpenClient("localhost:10018", false)
	test(t, c)

	node.Stop()

	cluster = "http://127.0.0.1:12378"

	node = NewKvNode()

	if err := node.Start(&id, &cluster); nil != err {
		panic(err)
	}

	//等到所有store都成为leader之后再发送指令
	waitCondition(func() bool {
		node.storeMgr.RLock()
		defer node.storeMgr.RUnlock()
		for _, v := range node.storeMgr.stores {
			if !v.rn.isLeader() {
				return false
			}
		}
		return true
	})

	node.Stop()

}

func TestKvnode1(t *testing.T) {

	//先删除所有kv文件
	os.RemoveAll("./kv-1-1")
	os.RemoveAll("./kv-1-1-snap")

	conf.LoadConfigStr(fmt.Sprintf(configStr, _sqltype, _host, _port, _user, _password, _db, _host, _port, _user, _password, _db))

	InitLogger()

	cluster := "http://127.0.0.1:12379"
	id := 1

	node := NewKvNode()

	if err := node.Start(&id, &cluster); nil != err {
		panic(err)
	}

	//等到所有store都成为leader之后再发送指令
	waitCondition(func() bool {
		node.storeMgr.RLock()
		defer node.storeMgr.RUnlock()
		for _, v := range node.storeMgr.stores {
			if !v.rn.isLeader() {
				return false
			}
		}
		return true
	})

	c := client.OpenClient("localhost:10018", false)
	test(t, c)

	for i := 0; i < 10; i++ {
		n := fmt.Sprintf("test:%d", i)
		c.GetAll("users1", n).Exec()
	}

	node.storeMgr.RLock()
	for _, v := range node.storeMgr.stores {
		v.doLRU()
	}
	node.storeMgr.RUnlock()

	for i := 0; i < 10; i++ {
		n := fmt.Sprintf("test:%d", i)
		c.Kick("users1", n).Exec()
	}

	//写入一个大字段触发快照压缩
	fields := map[string]interface{}{}
	fields["age"] = 12
	fields["name"] = "sniperHW"
	fields["phone"] = strings.Repeat("a", 4096)

	r1 := c.Set("users1", "sniperHW", fields).Exec()
	assert.Equal(t, errcode.ERR_OK, r1.ErrCode)

	node.storeMgr.RLock()
	for _, v := range node.storeMgr.stores {
		v.rn.triggerSnapshot()
	}
	node.storeMgr.RUnlock()

	node.Stop()

}
