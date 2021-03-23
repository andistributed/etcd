package etcdconfig

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// AutoSyncInterval is the interval to update endpoints with its latest members.
// 0 disables auto-sync. By default auto-sync is disabled.
func AutoSyncInterval(v time.Duration) Configer {
	return func(c *clientv3.Config) {
		c.AutoSyncInterval = v
	}
}

// DialKeepAliveTime is the time after which client pings the server to see if
// transport is alive.
func DialKeepAliveTime(v time.Duration) Configer {
	return func(c *clientv3.Config) {
		c.DialKeepAliveTime = v
	}
}

// DialKeepAliveTimeout is the time that the client waits for a response for the
// keep-alive probe. If the response is not received in this time, the connection is closed.
func DialKeepAliveTimeout(v time.Duration) Configer {
	return func(c *clientv3.Config) {
		c.DialKeepAliveTimeout = v
	}
}

// MaxCallSendMsgSize is the client-side request send limit in bytes.
// If 0, it defaults to 2.0 MiB (2 * 1024 * 1024).
// Make sure that "MaxCallSendMsgSize" < server-side default send/recv limit.
// ("--max-request-bytes" flag to etcd or "embed.Config.MaxRequestBytes").
func MaxCallSendMsgSize(v int) Configer {
	return func(c *clientv3.Config) {
		c.MaxCallSendMsgSize = v
	}
}

// MaxCallRecvMsgSize is the client-side response receive limit.
// If 0, it defaults to "math.MaxInt32", because range response can
// easily exceed request send limits.
// Make sure that "MaxCallRecvMsgSize" >= server-side default send/recv limit.
// ("--max-request-bytes" flag to etcd or "embed.Config.MaxRequestBytes").
func MaxCallRecvMsgSize(v int) Configer {
	return func(c *clientv3.Config) {
		c.MaxCallRecvMsgSize = v
	}
}

// TLS holds the client secure credentials, if any.
func TLS(v *tls.Config) Configer {
	return func(c *clientv3.Config) {
		c.TLS = v
	}
}

// Noop no operations
func Noop(c *clientv3.Config) {
	// Noop
}

// TLSFile TLS holds the client secure credentials, if any.
func TLSFile(etcdCertFile, etcdKeyFile string) Configer {
	if len(etcdCertFile) == 0 || len(etcdKeyFile) == 0 {
		return Noop
	}
	cert, err := ioutil.ReadFile(etcdCertFile)
	if err != nil {
		log.Fatal(err)
	}

	key, err := ioutil.ReadFile(etcdKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		log.Fatal(err)
	}
	return TLS(&tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{certificate},
	})
}

// Username is a user name for authentication.
func Username(v string) Configer {
	return func(c *clientv3.Config) {
		c.Username = v
	}
}

// Password is a password for authentication.
func Password(v string) Configer {
	return func(c *clientv3.Config) {
		c.Password = v
	}
}

// RejectOldCluster when set will refuse to create a client against an outdated cluster.
func RejectOldCluster(v bool) Configer {
	return func(c *clientv3.Config) {
		c.RejectOldCluster = v
	}
}

// DialOptions is a list of dial options for the grpc client (e.g., for interceptors).
func DialOptions(v []grpc.DialOption) Configer {
	return func(c *clientv3.Config) {
		c.DialOptions = v
	}
}

// Context is the default client context; it can be used to cancel grpc dial out and
// other operations that do not have an explicit context.
func Context(v context.Context) Configer {
	return func(c *clientv3.Config) {
		c.Context = v
	}
}
