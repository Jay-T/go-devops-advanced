package helpers

import (
	"context"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc/metadata"
)

// GetReqID helper returns Request-ID from metadata.
func GetReqID(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	return md.Get("Request-ID")[0]
}

// GetLocalInterfaceAddress returns IP address of interface <ifname>.
func GetLocalInterfaceAddress(remoteAddress string) (string, error) {
	var localAddress string

	conn, err := net.Dial("udp4", remoteAddress)
	if err != nil {
		log.Println(err)
		return "", err
	}

	defer func() {
		err = conn.Close()
		if err != nil {
			log.Print(err)
		}
	}()

	localAddress = strings.Split(conn.LocalAddr().String(), ":")[0]

	return localAddress, nil
}
