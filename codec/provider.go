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

func TcpDefaultProvider(pkgStructure func() DataPtr) Provider {
	return func(firstPkg []byte) (Name, Codec, PkgBuilder, []byte) {
		tag := firstPkg[0]
		firstPkg = firstPkg[1:]
		if tag == byte('j') {
			return Json, NewDelimiterCodec([]byte("\n\n"), []byte("\n\n")), NewJsonPackageBuilder(pkgStructure), firstPkg
		}

		return Proto, NewLengthCodec(0xAB, 1024), NewProtobufPackageBuilder(pkgStructure), firstPkg
	}
}

func WssDefaultProvider(pkgStructure func() DataPtr) Provider {
	return func(firstPkg []byte) (Name, Codec, PkgBuilder, []byte) {
		tag := firstPkg[0]
		firstPkg = firstPkg[1:]
		if tag == byte('j') {
			return Json, NewWebsocketCodec(), NewJsonPackageBuilder(pkgStructure), firstPkg
		}

		return Proto, NewWebsocketCodec(), NewProtobufPackageBuilder(pkgStructure), firstPkg
	}
}
