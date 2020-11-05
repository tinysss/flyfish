package main

import (
	"flag"
	"fmt"
	"github.com/sniperHW/flyfish/client"
	"github.com/sniperHW/flyfish/errcode"
)

func main() {
	serverAddr := flag.String("server-addr", "localhost:10012", "server address")
	compressFlag := flag.Bool("compress", false, "whether compress data")

	flag.Parse()

	var (
		table   = "user_module_data"
		key     = "4540217385795581"
		version = []int64{10}
		field   = "rankdata"
		old     = []byte{1}
		new     = []byte{0}
	)

	client := client.OpenClient(*serverAddr, *compressFlag)

	result := client.CompareAndSetNx(table, key, field, old, new, version...).Exec()
	if result.ErrCode == errcode.ERR_OK {
		fmt.Printf("compare-set-nx successfully, version=%d.\n", result.Version)
	} else {
		fmt.Printf("compare-set-nx failed, err: %s.\n", errcode.GetErrorStr(result.ErrCode))
	}
}