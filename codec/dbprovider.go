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
	return &Dbp{}
}

func (p *Dbp) Provider(name Name) DataBuilder {
	if name == Json {
		p.Do(func() {
			p.json = NewJsonDataBuilder()
		})
		return p.json
	}

	p.Do(func() {
		p.proto = NewProtobufDataBuilder()
	})
	return p.proto
}
