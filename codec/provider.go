package codec

const (
	Proto Name = "proto"
	Json  Name = "json"
)

type Name string

func (n Name) String() string {
	return string(n)
}

type Provider interface {
	ParseByPackage(firstPkg []byte) (Name, Codec, PkgBuilder, []byte)
	GetByName(name Name) (Name, Codec, PkgBuilder)
}

type TcpProvider struct {
	toData    func(p *PKG) DataPtr
	toPKG     func(d DataPtr) *PKG
	delimiter []byte
	magicNum  uint16
	bodyMax   int
}

func NewTcpProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) *TcpProvider {
	return &TcpProvider{toData: toData, toPKG: toPKG, delimiter: []byte("\\N\\B"), magicNum: 0xAB}
}

func (s *TcpProvider) SetDelimiter(delimiter []byte) {
	s.delimiter = delimiter
}

func (s *TcpProvider) SetMagicNum(magicNum uint16) {
	s.magicNum = magicNum
}

func (s *TcpProvider) SetBodyMax(bodyMax int) {
	s.bodyMax = bodyMax
}

func (s *TcpProvider) ParseByPackage(firstPkg []byte) (Name, Codec, PkgBuilder, []byte) {
	tag := firstPkg[0]
	if tag == byte('{') {
		return Json, NewDelimiterCodec(s.delimiter, s.delimiter), NewJsonPackageBuilder(s.toData, s.toPKG), firstPkg
	}
	if tag == byte('j') {
		return Json, NewDelimiterCodec(s.delimiter, s.delimiter), NewJsonPackageBuilder(s.toData, s.toPKG), firstPkg[1:]
	}

	return Proto, NewLengthCodec(s.magicNum, s.bodyMax), NewProtobufPackageBuilder(s.toData, s.toPKG), firstPkg
}

func (s *TcpProvider) GetByName(name Name) (Name, Codec, PkgBuilder) {
	if name == Json {
		return Json, NewDelimiterCodec(s.delimiter, s.delimiter), NewJsonPackageBuilder(s.toData, s.toPKG)
	}
	return Proto, NewLengthCodec(s.magicNum, s.bodyMax), NewProtobufPackageBuilder(s.toData, s.toPKG)
}

func TcpDefaultProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) Provider {
	return NewTcpProvider(toData, toPKG)
}

type WssProvider struct {
	toData func(p *PKG) DataPtr
	toPKG  func(d DataPtr) *PKG
}

func NewWssProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) *WssProvider {
	return &WssProvider{toData, toPKG}
}

func (s *WssProvider) ParseByPackage(firstPkg []byte) (Name, Codec, PkgBuilder, []byte) {
	tag := firstPkg[0]
	if tag == byte('{') {
		return Json, NewWebsocketCodec(), NewJsonPackageBuilder(s.toData, s.toPKG), firstPkg
	}
	if tag == byte('j') {
		return Json, NewWebsocketCodec(), NewJsonPackageBuilder(s.toData, s.toPKG), firstPkg[1:]
	}

	return Proto, NewWebsocketCodec(), NewProtobufPackageBuilder(s.toData, s.toPKG), firstPkg
}

func (s *WssProvider) GetByName(name Name) (Name, Codec, PkgBuilder) {
	if name == Json {
		return Json, NewWebsocketCodec(), NewJsonPackageBuilder(s.toData, s.toPKG)
	}
	return Proto, NewWebsocketCodec(), NewProtobufPackageBuilder(s.toData, s.toPKG)
}

func WssDefaultProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) Provider {
	return NewWssProvider(toData, toPKG)
}

func UdpDefaultProvider(toData func(p *PKG) DataPtr, toPKG func(d DataPtr) *PKG) Provider {
	return NewWssProvider(toData, toPKG)
}
