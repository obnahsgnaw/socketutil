package codec

const (
	Proto Name = "proto"
	Json  Name = "json"
)

type Name string

func (n Name) String() string {
	return string(n)
}

type Provider func(firstPkg []byte) (Name, Codec, PkgBuilder, []byte)

func TcpDefaultProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) Provider {
	return func(firstPkg []byte) (Name, Codec, PkgBuilder, []byte) {
		tag := firstPkg[0]
		if tag == byte('{') {
			return Json, NewDelimiterCodec([]byte("\\N\\B"), []byte("\\N\\B")), NewJsonPackageBuilder(toData, toPKG), firstPkg
		}
		if tag == byte('j') {
			return Json, NewDelimiterCodec([]byte("\\N\\B"), []byte("\\N\\B")), NewJsonPackageBuilder(toData, toPKG), firstPkg[1:]
		}

		return Proto, NewLengthCodec(0xAB, 1024), NewProtobufPackageBuilder(toData, toPKG), firstPkg
	}
}

func WssDefaultProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) Provider {
	return func(firstPkg []byte) (Name, Codec, PkgBuilder, []byte) {
		tag := firstPkg[0]
		if tag == byte('{') {
			return Json, NewWebsocketCodec(), NewJsonPackageBuilder(toData, toPKG), firstPkg
		}
		if tag == byte('j') {
			return Json, NewWebsocketCodec(), NewJsonPackageBuilder(toData, toPKG), firstPkg[1:]
		}

		return Proto, NewWebsocketCodec(), NewProtobufPackageBuilder(toData, toPKG), firstPkg
	}
}

func UdpDefaultProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) Provider {
	return func(firstPkg []byte) (Name, Codec, PkgBuilder, []byte) {
		tag := firstPkg[0]
		if tag == byte('{') {
			return Json, NewWebsocketCodec(), NewJsonPackageBuilder(toData, toPKG), firstPkg
		}
		if tag == byte('j') {
			return Json, NewWebsocketCodec(), NewJsonPackageBuilder(toData, toPKG), firstPkg[1:]
		}

		return Proto, NewWebsocketCodec(), NewProtobufPackageBuilder(toData, toPKG), firstPkg
	}
}
