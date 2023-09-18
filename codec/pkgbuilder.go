package codec

import (
	"encoding/json"
	"errors"
	"google.golang.org/protobuf/proto"
	"strconv"
)

//
//import (
//	"encoding/json"
//	"errors"
//	"google.golang.org/protobuf/proto"
//)

type ActionId uint32

func (a ActionId) String() string {
	return strconv.Itoa(int(a))
}

func (a ActionId) Val() uint32 {
	return uint32(a)
}

// Action id
type Action struct {
	Id   ActionId
	Name string
}

func (a Action) String() string {
	return a.Id.String() + ":" + a.Name
}

func NewAction(id ActionId, name string) Action {
	return Action{
		Id:   id,
		Name: name,
	}
}

// PkgBuilder 包构建器
type PkgBuilder interface {
	Unpack(b []byte) (p DataPtr, err error)
	Pack(p DataPtr) (b []byte, err error)
}

func packErr(err error) error {
	return errors.New("pkg builder error: Pack package failed, err=" + err.Error())
}

func unpackErr(err error) error {
	return errors.New("pkg builder error: Unpack package failed, err=" + err.Error())
}

var (
	ErrNoData  = errors.New("pkg builder error: Unpack package failed, err= no data. ")
	ErrDataNil = errors.New("pkg builder error: pack package failed, err= data is nil. ")
)

// ProtobufPackageBuilder protobuf 包构建器
type ProtobufPackageBuilder struct {
	gen func() DataPtr
}

// NewProtobufPackageBuilder return a protobuf package builder
func NewProtobufPackageBuilder(pgkGener func() DataPtr) *ProtobufPackageBuilder {
	return &ProtobufPackageBuilder{gen: pgkGener}
}

// Unpack 拆包
func (pp *ProtobufPackageBuilder) Unpack(b []byte) (p DataPtr, err error) {
	if len(b) == 0 {
		err = ErrNoData
		return
	}
	p1 := pp.gen()
	if p2, ok := p.(proto.Message); !ok {
		err = packErr(errors.New("not prot message"))
		return
	} else {
		if err = proto.Unmarshal(b, p2); err != nil {
			err = unpackErr(err)
		}
	}
	p = p1

	return
}

// Pack 封包
func (pp *ProtobufPackageBuilder) Pack(p DataPtr) (b []byte, err error) {
	if p == nil {
		err = ErrDataNil
		return
	}
	if p1, ok := p.(proto.Message); !ok {
		err = packErr(errors.New("not prot message"))
		return
	} else {
		if b, err = proto.Marshal(p1); err != nil {
			err = packErr(err)
		}
	}

	return
}

type JsonPackageBuilder struct {
	gen func() DataPtr
}

func NewJsonPackageBuilder(pgkGener func() DataPtr) *JsonPackageBuilder {
	return &JsonPackageBuilder{gen: pgkGener}
}

// Unpack 拆包
func (pp *JsonPackageBuilder) Unpack(b []byte) (p DataPtr, err error) {
	if len(b) == 0 {
		err = ErrNoData
		return
	}
	p1 := pp.gen()
	if err = json.Unmarshal(b, &p1); err != nil {
		err = unpackErr(err)
	}

	p = p1

	return
}

// Pack 封包
func (pp *JsonPackageBuilder) Pack(p DataPtr) (b []byte, err error) {
	if p == nil {
		err = ErrDataNil
		return
	}
	if b, err = json.Marshal(p); err != nil {
		err = packErr(err)
	}

	return
}
