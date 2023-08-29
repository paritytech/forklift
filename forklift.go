package main

import (
	"encoding/binary"
	"forklift/Commands"
)

func main() {

	//os.Setenv("S3_ACCESS_KEY_ID", "pLwDdUcVhCKNqXDEH4ze")
	//os.Setenv("S3_SECRET_ACCESS_KEY", "qvtkoH5m1hA3ZCHuTMD1RlbOeuVGdvsSlD8bE8lZ")
	//os.Setenv("S3_ENDPOINT_URL", "192.168.1.2:9000")
	//os.Setenv("S3_BUCKET_NAME", "forklift")

	Commands.Execute()

	return
}

func Int64ToByteArray(num uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, num)

	return b
}

func Int32ToByteArray(num uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, num)

	return b
}
