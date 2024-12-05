package openziti

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openziti/sdk-golang/ziti"
)

var ZitiContext ziti.Context
var ZitiContexts *ziti.CtxCollection
var ZitiTransport *http.Transport
var ZitiClient *http.Client
var ZitiCustomDialer CustomDialer

// CustomDialer wraps a ziti.Context
// Needed when can't use ziti transport(ex. nats)
type CustomDialer struct {
	ZitiContext ziti.Context
}

func (cd CustomDialer) Dial(network, address string) (net.Conn, error) {
	addr := strings.Split(address, ":")
	return cd.ZitiContext.Dial(addr[0])
}

type FallbackDialer struct {
	UnderlayDialer *net.Dialer
}

func (z FallbackDialer) Dial(network, address string) (net.Conn, error) {
	return z.UnderlayDialer.Dial(network, address)
}

// Enrolls api identity if needed
// Creates openziti api session
// Creates a keepalive goroutine which pings openziti api every X minutes(default 10)
func SetupOpenziti(ctrlUrl string, zitiIDPath string) error {
	err := EnrollIfNeeded(zitiIDPath)
	if err != nil {
		log.Print(err)
		return err
	}
	err = CreateApiSession(ctrlUrl, zitiIDPath)
	if err != nil {
		log.Print(err)
		return err
	}
	err = InitCon(zitiIDPath)
	if err != nil {
		log.Print(err)
		return err
	}

	go KeepAlive(ctrlUrl)
	return nil
}

func InitCon(zitiIDPath string) error {
	err := SetupZitiContext(zitiIDPath)
	if err != nil {
		log.Print(err)
		return err
	}
	err = SetupZitiTransport()
	if err != nil {
		log.Print(err)
		return err
	}
	ZitiClient = &http.Client{
		Transport: ZitiTransport,
		Timeout:   30 * time.Second,
	}
	ZitiCustomDialer = CustomDialer{ZitiContext: ZitiContext}

	// Set ziti transport for websocket
	websocket.ZitiTransport = ZitiTransport
	return nil
}

func SetupZitiContext(path string) (err error) {
	identityFile := path + ".json"

	cfg, err := ziti.NewConfigFromFile(identityFile)
	if err != nil {
		log.Print(err)
		return err
	}
	cfg.ConfigTypes = append(cfg.ConfigTypes, "all")

	ZitiContext, err = ziti.NewContext(cfg)
	if err != nil {
		log.Print(err)
		return err
	}
	_ = ZitiContext.RefreshServices()
	return nil
}

func SetupZitiTransport() error {
	ZitiContexts = ziti.NewSdkCollection()
	ZitiContexts.Add(ZitiContext)
	fallback := &FallbackDialer{
		UnderlayDialer: &net.Dialer{},
	}

	ZitiTransport = http.DefaultTransport.(*http.Transport).Clone() // copy default transport
	ZitiTransport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := ZitiContexts.NewDialerWithFallback(ctx, fallback)
		return dialer.Dial(network, addr)
	}
	ZitiTransport.Dial = func(network, addr string) (net.Conn, error) {
		dialer := ZitiContexts.NewDialerWithFallback(context.Background(), fallback)
		return dialer.Dial(network, addr)
	}
	return nil
}
