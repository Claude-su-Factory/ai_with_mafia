package ai

import (
	"math/rand"

	"ai-playground/config"
)

type Persona struct {
	Name        string
	Personality string
}

var defaultPersonas = []Persona{
	{"김민준", "논리적이고 말이 많다. 항상 근거를 들어 주장한다."},
	{"이지은", "조용하지만 날카롭다. 말수는 적지만 핵심을 찌른다."},
	{"박서연", "감정적으로 반응한다. 직관을 믿고 직설적으로 표현한다."},
	{"최도현", "의심이 많다. 모든 발언에서 모순을 찾으려 한다."},
	{"정하은", "친근하고 사교적이다. 갈등을 중재하려 하지만 때로 우유부단하다."},
	{"강태양", "대담하고 공격적이다. 선제적으로 의심을 제기하며 주도권을 잡으려 한다."},
	{"윤서희", "차분하고 관찰력이 뛰어나다. 다른 사람의 행동 패턴을 분석한다."},
	{"임재현", "유머러스하다. 농담으로 분위기를 바꾸려 하며 진지한 상황을 회피하기도 한다."},
}

type PersonaPool struct {
	personas []Persona
}

func NewPersonaPool(cfgPersonas []config.PersonaConfig) *PersonaPool {
	if len(cfgPersonas) == 0 {
		return &PersonaPool{personas: defaultPersonas}
	}
	ps := make([]Persona, len(cfgPersonas))
	for i, p := range cfgPersonas {
		ps[i] = Persona{Name: p.Name, Personality: p.Personality}
	}
	return &PersonaPool{personas: ps}
}

// Assign returns n unique personas (cycles if pool is smaller than n).
func (p *PersonaPool) Assign(n int) []Persona {
	pool := make([]Persona, len(p.personas))
	copy(pool, p.personas)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	result := make([]Persona, n)
	for i := range result {
		result[i] = pool[i%len(pool)]
	}
	return result
}
