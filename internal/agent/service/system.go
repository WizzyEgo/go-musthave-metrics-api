package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
)

// cpuSampleInterval — окно, за которое считается загрузка каждого CPU.
// Ненулевой интервал даёт значения уже на первом опросе.
const cpuSampleInterval = time.Second

func (s *Service) collectSystem(ctx context.Context) {
	s.collectMemory(ctx)
	s.collectCPU(ctx)
}

func (s *Service) collectMemory(ctx context.Context) {
	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		s.logCollectErr("failed to read memory stats", err)
		return
	}
	s.metrics.SetGauge("TotalMemory", float64(vm.Total))
	s.metrics.SetGauge("FreeMemory", float64(vm.Free))
}

func (s *Service) collectCPU(ctx context.Context) {
	percents, err := cpu.PercentWithContext(ctx, cpuSampleInterval, true)
	if err != nil {
		s.logCollectErr("failed to read cpu stats", err)
		return
	}
	for i, percent := range percents {
		s.metrics.SetGauge(fmt.Sprintf("CPUutilization%d", i+1), percent)
	}
}

func (s *Service) logCollectErr(msg string, err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	s.log.Error(msg, zap.Error(err))
}
