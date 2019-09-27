package kvnode

import (
	"fmt"
	pb "github.com/golang/protobuf/proto"
	codec "github.com/sniperHW/flyfish/codec"
	"github.com/sniperHW/flyfish/dbmeta"
	"github.com/sniperHW/flyfish/errcode"
	"github.com/sniperHW/flyfish/proto"
	"github.com/sniperHW/kendynet"
	"time"
)

type opDel struct {
	kv       *kv
	deadline time.Time
	replyer  *replyer
	seqno    int64
	version  *int64
}

func (this *opDel) reply(errCode int32, fields map[string]*proto.Field, version int64) {
	this.replyer.reply(this, errCode, fields, version)
}

func (this *opDel) dontReply() {
	this.replyer.dontReply()
}

func (this *opDel) causeWriteBack() bool {
	return true
}

func (this *opDel) isSetOp() bool {
	return true
}

func (this *opDel) isReplyerClosed() bool {
	this.replyer.isClosed()
}

func (this *opDel) getKV() *kv {
	return this.kv
}

func (this *opDel) isTimeout() bool {
	return time.Now().After(this.deadline)
}

func (this *opDel) makeResponse(errCode int32, fields map[string]*proto.Field, version int64) pb.Message {

	var key string

	if nil != this.kv {
		key = this.kv.key
	}

	return &proto.DelResp{
		Head: &proto.RespCommon{
			Key:     pb.String(key),
			Seqno:   pb.Int64(this.seqno),
			ErrCode: pb.Int32(errCode),
			Version: pb.Int64(version),
		},
	}
}

func del(n *kvnode, session kendynet.StreamSession, msg *codec.Message) {

	req := msg.GetData().(*proto.DelReq)

	head := req.GetHead()

	head := req.GetHead()
	op := &opDel{
		deadline: time.Now().Add(time.Duration(head.GetTimeout())),
		replyer:  newReplyer(session, time.Now().Add(time.Duration(head.GetRespTimeout()))),
		seqno:    head.GetSeqno(),
		version:  req.Version,
	}

	err := checkReqCommon(head)

	if err != errcode.ERR_OK {
		op.reply(err, nil, -1)
		return
	}

	kv, _ := n.storeMgr.getkv(head.GetTable(), head.GetKey())

	if nil == kv {
		op.reply(errcode.ERR_INVAILD_TABLE, nil, -1)
		return
	}

	op.kv = kv

	if !kv.opQueue.append(op) {
		op.reply(errcode.ERR_BUSY, nil, -1)
		return
	}

	kv.processQueueOp()
}
