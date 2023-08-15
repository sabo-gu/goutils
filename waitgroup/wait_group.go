package waitgroup

import (
	"fmt"
	"io"
	"sync"

	"go.uber.org/multierr"

	"github.com/DoOR-Team/goutils/log"
)

type Srv struct {
	CloseFunc func() error
	ServeFunc func() error
}

func (h *Srv) Close() error {
	return h.CloseFunc()
}

func (h *Srv) Serve() error {
	return h.ServeFunc()
}

type Cli struct {
	CloseFunc func() error
}

func (c *Cli) Close() error {
	return c.CloseFunc()
}

type Mod interface {
	io.Closer
}

type Client interface {
	Mod
}

type Server interface {
	Mod
	//阻塞运行，返回错误将会结束进程，否则什么都不做
	Serve() error
}

type NoopMod struct {
}

func (*NoopMod) Close() error {
	return nil
}

var defaultWaitGroupWrapper = waitGroupWrapper{
	mods: make(map[string]Mod),
}

func (w *waitGroupWrapper) InitModsAndWrapServersWithExitFunc(exit func()) error {
	if w.closed {
		log.Error("[waitgroup] waitgroup已经关闭，无法启动")
		return fmt.Errorf("[waitgroup] Cant init after closed")
	}
	if w.inited {
		log.Error("[waitgroup] waitgroup已经被初始化")
		return fmt.Errorf("[waitgroup] Cant init mods twice")
	}

	w.exit = exit

	sorts := sortCreators(true)
	for _, cw := range sorts {
		name := cw.name
		mod := cw.c()
		if mod == nil {
			log.Error("[waitgroup] 模块获取失败")
			return fmt.Errorf("[waitgroup] Can't returns nil mod")
		}
		log.Notice("[waitgroup] 初始化模块：", name)
		err := w.addModAndWrapServer(name, mod)
		if err != nil {
			return err
		}
	}

	w.inited = true
	return nil
}

type waitGroupWrapper struct {
	sync.WaitGroup

	mods  map[string]Mod
	names []string

	inited bool
	closed bool

	exit func()
}

func InitModsAndWrapServersWithExitFunc(exit func()) error {
	err := defaultWaitGroupWrapper.InitModsAndWrapServersWithExitFunc(exit)
	if err != nil {
		panic(err)
	}
	return err
}

func (w *waitGroupWrapper) AddModAndWrapServer(name string, mod Mod) error {
	return w.addModAndWrapServer(name, mod)
}

func (w *waitGroupWrapper) addModAndWrapServer(name string, mod Mod) error {
	if _, ok := w.mods[name]; ok {
		return fmt.Errorf("[wait group] Mod %s already exists", name)
	}

	w.mods[name] = mod
	w.names = append(w.names, name)

	svc, ok := mod.(Server)
	if ok {
		w.Wrap(func() {
			log.Infof("[wait group] 模块 %s 初始化成功", name)
			if err := svc.Serve(); err != nil {
				//如果还未closed就执行外界提供的结束进程方法
				//这个主要是为了防止死锁，很多服务的Close方法执行后这里会抛一个错误出来
				//然后再调用exit，exit里调用CloseServices内部就会死锁
				if !w.closed && w.exit != nil {
					log.Errorf("[wait group] 模块 %s 初始化错误 err: %v", name, err)
					w.exit()
				}
			}
		})
	}

	return nil
}

func (w *waitGroupWrapper) CloseMods() error {
	if w.closed {
		return fmt.Errorf("[wait group] Mods is already closed")
	}

	if !w.inited {
		return fmt.Errorf("[wait group] Please init first")
	}
	w.closed = true

	var errs error
	for index := len(w.names) - 1; index >= 0; index-- {
		name := w.names[index]
		mod, ok := w.mods[name]
		if !ok {
			continue
		}

		err := mod.Close()
		if err != nil {
			errs = multierr.Append(errs, err)
			log.Errorf("[wait group] %s 模块关闭失败 - %v", name, err)
		} else {
			log.Infof("[wait group] %s mod closed", name)
		}
	}
	return errs
}

func (w *waitGroupWrapper) Wrap(cb func()) {
	w.Add(1)
	go func() {
		cb()
		w.Done()
	}()
}

func CloseModsAndWait() error {
	err := defaultWaitGroupWrapper.CloseMods()
	defaultWaitGroupWrapper.Wait()
	return err
}

func AddModAndWrapServer(name string, mod Mod) error {
	return defaultWaitGroupWrapper.AddModAndWrapServer(name, mod)
}
