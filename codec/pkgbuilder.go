package codec

import (
	"encoding/json"
	"errors"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNoData  = errors.New("pkg builder error: Unpack package failed, err= no data. ")
	ErrDataNil = errors.New("pkg builder error: pack package failed, err= data is nil. ")
)

func packErr(err error) error {
	return errors.New("pkg builder error: Pack package failed, err=" + err.Error())
}

func unpackErr(err error) error {
	return errors.New("pkg builder error: Unpack package failed, err=" + err.Error())
}

type PKG struct {
	Action ActionId
	Data   []byte
}

// PkgBuilder 包构建器
type PkgBuilder interface {
	Unpack(b []byte) (p *PKG, err error)
	Pack(p *PKG) (b []byte, err error)
}

// ProtobufPackageBuilder protobuf 包构建器
type ProtobufPackageBuilder struct {
	gen func(*PKG) DataPtr
	to  func(DataPtr) *PKG
}

// NewProtobufPackageBuilder return a protobuf package builder, toData *PKG可为nil
func NewProtobufPackageBuilder(toData func(*PKG) DataPtr, toPKG func(DataPtr) *PKG) *ProtobufPackageBuilder {
	return &ProtobufPackageBuilder{gen: toData, to: toPKG}
}

// Unpack 拆包
func (pp *ProtobufPackageBuilder) Unpack(b []byte) (p *PKG, err error) {
	if len(b) == 0 {
		err = ErrNoData
		return
	}
	p1 := pp.gen(&PKG{})
	if p2, ok := p1.(proto.Message); !ok {
		err = packErr(errors.New("not proto message"))
		return
	} else {
		if err = proto.Unmarshal(b, p2); err != nil {
			err = unpackErr(err)
		}
	}
	p = pp.to(p1)

	return
}

// Pack 封包
func (pp *ProtobufPackageBuilder) Pack(p *PKG) (b []byte, err error) {
	if p == nil {
		err = ErrDataNil
		return
	}
	p2 := pp.gen(p)
	if p1, ok := p2.(proto.Message); !ok {
		err = packErr(errors.New("not proto message"))
		return
	} else {
		if b, err = proto.Marshal(p1); err != nil {
			err = packErr(err)
		}
	}

	return
}

type JsonPackageBuilder struct {
	gen func(*PKG) DataPtr
	to  func(DataPtr) *PKG
}

func NewJsonPackageBuilder(toData func(*PKG) DataPtr, toPKG func(DataPtr) *PKG) *JsonPackageBuilder {
	return &JsonPackageBuilder{gen: toData, to: toPKG}
}

// Unpack 拆包
func (pp *JsonPackageBuilder) Unpack(b []byte) (p *PKG, err error) {
	if len(b) == 0 {
		err = ErrNoData
		return
	}
	p1 := pp.gen(&PKG{})
	if p1 == nil {
		err = unpackErr(errors.New("unpack to data is nil"))
		return
	}
	if err = json.Unmarshal(b, p1); err != nil {
		err = unpackErr(err)
	}

	p = pp.to(p1)

	return
}

// Pack 封包
func (pp *JsonPackageBuilder) Pack(p *PKG) (b []byte, err error) {
	if p == nil {
		err = ErrDataNil
		return
	}
	p1 := pp.gen(p)
	if p1 == nil {
		err = unpackErr(errors.New("pack to data is nil"))
		return
	}
	if b, err = json.Marshal(p1); err != nil {
		err = packErr(err)
	}

	return
}
