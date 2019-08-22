package flyfish

import (
	"github.com/sniperHW/flyfish/errcode"
	//"github.com/sniperHW/flyfish/proto"
)

type sqlResponseLocalCache struct {
}

func (this sqlResponseLocalCache) onSqlNotFound(ctx *processContext) {
	Debugln("onSqlNotFound key", ctx.getUniKey())
	cmdType := ctx.getCmdType()
	ckey := ctx.getCacheKey()
	if cmdType == cmdGet || cmdType == cmdDel || cmdType == cmdCompareAndSet {
		ctx.reply(errcode.ERR_NOTFOUND, nil, -1)
		ckey.setMissing()
		ckey.processQueueCmd()
	} else {
		ctx.writeBackFlag = write_back_insert
		ckey.unit.doWriteBack(ctx)
	}
}

func (this sqlResponseLocalCache) onSqlLoadOKGet(ctx *processContext) {

	Debugln("onSqlLoadOKGet")

	version := ctx.fields["__version__"].GetInt()
	ckey := ctx.getCacheKey()
	ckey.mtx.Lock()
	ckey.setValueNoLock(ctx)
	ckey.setOKNoLock(version)
	ckey.mtx.Unlock()
	ctx.reply(errcode.ERR_OK, ctx.fields, version)
	ckey.processQueueCmd()
}

func (this sqlResponseLocalCache) onSqlLoadOKSet(ctx *processContext) {

	Debugln("onSqlLoadOKSet")

	version := ctx.fields["__version__"].GetInt()

	Debugln("onSqlLoadOKSet", version)

	cmd := ctx.getCmd()
	ckey := ctx.getCacheKey()
	ckey.mtx.Lock()
	ckey.setValueNoLock(ctx)
	ckey.setOKNoLock(version)
	ckey.mtx.Unlock()

	cmdType := cmd.cmdType
	ctx.writeBackFlag = write_back_none
	if cmdType == cmdSet {
		if nil != cmd.version && *cmd.version != version {
			//版本号不对
			ctx.reply(errcode.ERR_VERSION, nil, version)
		} else {
			ctx.writeBackFlag = write_back_update //sql中存在,使用update回写
		}
	} else if cmdType == cmdCompareAndSet || cmdType == cmdCompareAndSetNx {
		dbV := ctx.fields[cmd.cns.oldV.GetName()]
		if !dbV.Equal(cmd.cns.oldV) {
			ctx.reply(errcode.ERR_NOT_EQUAL, ctx.fields, version)
		} else {
			ctx.fields[cmd.cns.oldV.GetName()] = cmd.cns.newV
			ctx.writeBackFlag = write_back_update //sql中存在,使用update回写
		}
	} else if cmdType == cmdSetNx {
		ctx.reply(errcode.ERR_KEY_EXIST, nil, version)
	} else {
		//cmdIncrBy/cmdDecrBy
		ctx.writeBackFlag = write_back_update //sql中存在,使用update回写
	}

	if ctx.writeBackFlag != write_back_none {
		ckey.unit.doWriteBack(ctx)
	} else {
		ckey.processQueueCmd()
	}
}

func (this sqlResponseLocalCache) onSqlLoadOKDel(ctx *processContext) {

	Debugln("onSqlLoadOKDel")

	var errCode int32
	version := ctx.fields["__version__"].GetInt()
	cmd := ctx.getCmd()
	ckey := ctx.getCacheKey()

	if nil != cmd.version && *cmd.version != version {
		//版本号不对
		errCode = errcode.ERR_VERSION
	} else {
		ctx.writeBackFlag = write_back_delete
		errCode = errcode.ERR_OK
	}

	ctx.reply(errCode, nil, version)

	if errCode == errcode.ERR_OK {
		//ckey.setMissing()
		ckey.unit.doWriteBack(ctx)
	} else {
		ckey.mtx.Lock()
		ckey.setValueNoLock(ctx)
		ckey.setOKNoLock(version)
		ckey.mtx.Unlock()
	}

	if ctx.writeBackFlag == write_back_none {
		ckey.processQueueCmd()
	}
}

func (this sqlResponseLocalCache) onSqlLoadOK(ctx *processContext) {
	Debugln("onSqlLoadOK")
	cmdType := ctx.getCmdType()
	if cmdType == cmdGet {
		this.onSqlLoadOKGet(ctx)
	} else if isSetCmd(cmdType) {
		this.onSqlLoadOKSet(ctx)
	} else if cmdType == cmdDel {
		this.onSqlLoadOKDel(ctx)
	} else {
		//记录日志
	}
}

func (this sqlResponseLocalCache) onSqlResp(ctx *processContext, errno int32) {
	Debugln("onSqlResp", ctx.getUniKey(), ctx.getCmdType(), errno)
	if errno == errcode.ERR_OK {
		this.onSqlLoadOK(ctx)
	} else if errno == errcode.ERR_NOTFOUND {
		this.onSqlNotFound(ctx)
	} else {
		onSqlExecError(ctx)
	}
}