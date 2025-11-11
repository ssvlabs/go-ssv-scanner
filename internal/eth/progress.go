package eth

import "sync"

// ScanProgress holds shared progress info to be read by a periodic ticker.
type ScanProgress struct {
    mu     sync.RWMutex
    Name   string
    From   int64
    To     int64
    Base   int64
    Latest int64
    Window int64
    Total  int64
    Done   int64
}

func (p *ScanProgress) SetName(name string) {
	p.mu.Lock()
	p.Name = name
	p.mu.Unlock()
}

func (p *ScanProgress) Update(from, to, latest, window int64) {
    p.mu.Lock()
    p.From = from
    p.To = to
    p.Latest = latest
    p.Window = window
    p.mu.Unlock()
}

func (p *ScanProgress) Snapshot() (name string, from, to, latest, base, window, total, done int64) {
    p.mu.RLock()
    name, from, to, latest, base, window, total, done = p.Name, p.From, p.To, p.Latest, p.Base, p.Window, p.Total, p.Done
    p.mu.RUnlock()
    return
}

func (p *ScanProgress) SetBase(base int64) {
    p.mu.Lock()
    p.Base = base
    p.mu.Unlock()
}

func (p *ScanProgress) SetTotal(total int64) {
    p.mu.Lock()
    p.Total = total
    p.Done = 0
    p.mu.Unlock()
}

func (p *ScanProgress) AddDone(delta int64) {
    if delta <= 0 { return }
    p.mu.Lock()
    p.Done += delta
    if p.Done > p.Total && p.Total > 0 {
        p.Done = p.Total
    }
    p.mu.Unlock()
}

func (p *ScanProgress) SetDone(done int64) {
    p.mu.Lock()
    p.Done = done
    if p.Done > p.Total && p.Total > 0 {
        p.Done = p.Total
    }
    p.mu.Unlock()
}
