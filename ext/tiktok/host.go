package tiktok

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

var apiHosts = []string{
	"api.us.tiktokv.com",
	"api15-normal-probe-useast2a.tiktokv.com",
	"api15-va.tiktokv.com",
	"api16-normal-apix-quic.tiktokv.com",
	"api16-normal-baseline.tiktokv.com",
	"api16-normal-c.tiktokv.com",
	"api16-normal-no1a.tiktokv.eu",
	"api16-normal-probe-useast2a.tiktokv.com",
	"api16-normal-quic-useast1a.tiktokv.com",
	"api16-normal-useast1a.tiktokv.com",
	"api16-normal-useast2a.tiktokv.com",
	"api16-normal-useast5.tiktokv.us",
	"api16-normal-useast5.us.tiktokv.com",
	"api16-normal-v4.tiktokv.com",
	"api17-normal-probe-useast2a.tiktokv.com",
	"api19-normal-c4-alisg.tiktokv.com",
	"api19-normal-probe-useast2a.tiktokv.com",
	"api19-normal-useast1a.tiktokv.com",
	"api19-normal.tiktokv.com",
	"api21-normal-c-useast2a.tiktokv.com",
	"api22-normal-c.tiktokv.com",
	"api22-normal-probe-useast2a.tiktokv.com",
	"api22-normal-v4.tiktokv.com",
	"api3-normal-useast1a.tiktokv.com",
	"api3-normal-useast2a.tiktokv.com",
	"api31-normal-probe-useast2a.tiktokv.com",
	"api31-normal-useast1a.tiktokv.com",
	"api32-normal-no1a.tiktokv.eu",
	"api32-normal-useast2a.tiktokv.com",
	"api58-normal-useast2a.tiktokv.com",
	"api9-normal-useast2a.tiktokv.com",
}

func GetRandomAPIHost() string {
	if len(apiHosts) == 0 {
		return ""
	}
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return apiHosts[time.Now().UnixNano()%int64(len(apiHosts))]
	}
	randInt := binary.BigEndian.Uint64(b)
	return apiHosts[randInt%uint64(len(apiHosts))]
}
