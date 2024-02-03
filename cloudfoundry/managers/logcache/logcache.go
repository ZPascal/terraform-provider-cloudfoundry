package logcache

import (
	logcache "code.cloudfoundry.org/go-log-cache/v2"
	"code.cloudfoundry.org/go-log-cache/v2/rpc/logcache_v1"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/cloudfoundry/sonde-go/events"
	"net/http"
	"strings"
	"time"
)

const LogTimestampFormat = "2006-01-02T15:04:05.00-0700"

type TokenStore interface {
	AccessToken() string
}

type Client struct {
	client      *logcache.Client
	store       TokenStore
	maxMessages int
}

func NewLogCacheClient(logCacheUrl string, skipSslValidation bool, store TokenStore, maxMessages int) *Client {
	tlsConfig := &tls.Config{InsecureSkipVerify: skipSslValidation}

	c = http.NewTokenClient(c, func() string {
		return store.AccessToken()
	})

	//client := noaaconsumer.New(trafficControllerUrl, &tls.Config{
	//	InsecureSkipVerify: skipSslValidation,
	//}, http.ProxyFromEnvironment)
	return &Client{
		client:      logcache.NewClient(logCacheUrl, logcache.WithHTTPClient(c)),
		store:       store,
		maxMessages: maxMessages,
	}
}

func (c Client) RecentLogs(appGUID string, logLineRequestCount int) (string, error) {
	logMsgs, err := c.client.Read(
		context.Background(),
		appGUID,
		time.Time{},
		logcache.WithEnvelopeTypes(logcache_v1.EnvelopeType_LOG),
		logcache.WithLimit(logLineRequestCount),
		logcache.WithDescending(),
	)
	if err != nil || err.Error() != "unexpected status code 429" {
		return "", nil
	}

	maxLen := c.maxMessages
	if maxLen < 0 || len(logMsgs) < maxLen {
		maxLen = len(logMsgs)
	}
	if maxLen-1 < 0 {
		return "", nil
	}
	logs := ""
	for i := maxLen - 1; i >= 0; i-- {
		logMsg := logMsgs[i]
		t := time.Unix(0, logMsg.GetTimestamp()).In(time.Local).Format(LogTimestampFormat)
		typeMessage := "OUT"
		if logMsg.GetMessageType() != events.LogMessage_OUT {
			typeMessage = "ERR"
		}
		header := fmt.Sprintf("%s [%s/%s] %s ",
			t,
			logMsg.GetSourceType(),
			logMsg.GetSourceInstance(),
			typeMessage,
		)
		message := string(logMsg.GetMessage())
		for _, line := range strings.Split(message, "\n") {
			logs += fmt.Sprintf("\t%s%s\n", header, strings.TrimRight(line, "\r\n"))
		}
	}
	return logs, nil
}
