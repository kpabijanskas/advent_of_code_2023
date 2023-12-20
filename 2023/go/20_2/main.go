package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

type pulse uint8

const (
	lowPulse pulse = iota
	highPulse
)

type moduleType uint8

const (
	mtBroadcaster moduleType = iota
	mtFlipFlip
	mtConjunction
)

type module struct {
	name       string
	t          moduleType
	flipFlopOn bool
	conjStates map[string]pulse
	dsts       []string
}

func (m module) shouldConjSendLow() bool {
	for _, s := range m.conjStates {
		if s == lowPulse {
			return false
		}
	}
	return true
}

func (m module) isConjAllLow() bool {
	for _, s := range m.conjStates {
		if s == highPulse {
			return false
		}
	}
	return true
}

type pulseMsg struct {
	src string
	pt  pulse
	dst string
}

func main() {
	data, err := os.ReadFile("./input")
	if err != nil {
		panic(err)
	}

	data = bytes.TrimSpace(data)
	br := bytes.NewReader(data)
	s := bufio.NewScanner(br)
	s.Split(bufio.ScanLines)

	// create modules
	modules := make(map[string]*module, bytes.Count(data, []byte("\n"))+1)
	for s.Scan() {
		var m module
		m.conjStates = make(map[string]pulse)

		fields := bytes.Split(s.Bytes(), []byte(" -> "))

		switch fields[0][0] {
		case '%':
			m.t = mtFlipFlip
			m.name = string(fields[0][1:])
		case '&':
			m.t = mtConjunction
			m.name = string(fields[0][1:])
		default:
			m.t = mtBroadcaster
			m.name = string(fields[0])
		}

		dsts := bytes.Split(fields[1], []byte(", "))
		m.dsts = make([]string, 0, len(dsts))
		for _, dst := range dsts {
			m.dsts = append(m.dsts, string(dst))
		}

		modules[m.name] = &m
	}

	// add conjunction module inputs
	for _, module := range modules {
		for _, target := range module.dsts {
			if mdl, ok := modules[target]; ok && mdl.t == mtConjunction {
				modules[target].conjStates[module.name] = lowPulse
			}
		}
	}

	// get conjunction modules that send to rx
	type loopStates struct {
		bps             uint64
		includedAllHigh bool
		finished        bool
	}
	loops := map[string]*loopStates{}
	var moduleBeforeRx string

LOOP:
	for _, module := range modules {
		for _, target := range module.dsts {
			if target == "rx" {
				moduleBeforeRx = module.name
				break LOOP
			}
		}
	}
	for input := range modules[moduleBeforeRx].conjStates {
		loops[input] = &loopStates{}
	}

	if moduleBeforeRx == "" {
		panic("did not find module before rx")
	}

	// run
	var buttonPresses uint
	var loopsFound int

LOOP2:
	for {
		buttonPresses++
		queue := []pulseMsg{}
		queue = append(queue, pulseMsg{
			pt:  lowPulse,
			dst: "broadcaster",
			src: "BUTTON",
		})

		for len(queue) > 0 {
			msg := queue[0]
			queue = queue[1:]

			if loopsFound == len(loops) {
				break LOOP2
			}
			mdl, ok := modules[msg.dst]
			// untyped module
			if !ok {
				continue
			}

			switch mdl.t {
			case mtBroadcaster:
				for _, dst := range mdl.dsts {
					queue = append(queue, pulseMsg{
						pt:  msg.pt,
						dst: dst,
						src: mdl.name,
					})
				}
			case mtFlipFlip:
				nextType := highPulse
				if msg.pt == highPulse {
					continue
				} else if mdl.flipFlopOn {
					nextType = lowPulse
				}

				for _, dst := range mdl.dsts {
					queue = append(queue, pulseMsg{
						pt:  nextType,
						dst: dst,
						src: mdl.name,
					})
				}

				mdl.flipFlopOn = !mdl.flipFlopOn
			case mtConjunction:
				if s, ok := loops[mdl.name]; ok && !s.finished {
					if s.includedAllHigh && mdl.isConjAllLow() {
						s.finished = true
						s.bps = uint64(buttonPresses)
						loopsFound++
					} else {
					}
				}

				mdl.conjStates[msg.src] = msg.pt
				nextType := highPulse
				if mdl.shouldConjSendLow() {
					if s, ok := loops[mdl.name]; ok && !s.includedAllHigh && !s.finished {
						s.includedAllHigh = true
					}
					nextType = lowPulse
				}

				for _, dst := range mdl.dsts {
					queue = append(queue, pulseMsg{
						pt:  nextType,
						dst: dst,
						src: mdl.name,
					})
				}
			default:
				panic("unknown module type")
			}
		}

	}

	counts := make([]uint64, 0, len(loops))
	for _, c := range loops {
		counts = append(counts, uint64(c.bps))
	}

	ans := counts[0]
	for _, c := range counts[1:] {
		ans = ans * c / gcd(ans, c)
	}

	fmt.Println(ans)
}

func gcd(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
