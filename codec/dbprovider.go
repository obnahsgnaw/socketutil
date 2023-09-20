package codec

import "sync"

type DataBuilderProvider interface {
	Provider(Name) DataBuilder
}

type Dbp struct {
	json  DataBuilder
	proto DataBuilder
	sync.Once
}

func NewDbp() *Dbp {
	return &Dbp{
		json:  NewJsonDataBuilder(),
		proto: NewProtobufDataBuilder(),
	}
}

func (p *Dbp) Provider(name Name) DataBuilder {
	if name == Json {
		return p.json
	}

	return p.proto
}
