package client

type PkgInterceptor interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

func GatewayPkgInterceptor(i PkgInterceptor) Option {
	return func(c *Client) {
		if i != nil {
			c.pkgInterceptor = i
		}
	}
}

func ListenInterceptor(listenInterceptor func([]byte) []byte) Option {
	return func(c *Client) {
		if listenInterceptor != nil {
			c.listenInterceptor = listenInterceptor
		}
	}
}
