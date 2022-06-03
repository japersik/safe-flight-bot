package flyPlanner

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/japersik/safe-flight-bot/model"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const updateNotifyTime = 10 * time.Second

type Planner interface {
	SetNotifier(notifier Notifier)
	PlanFly(info model.FlyPlan) (flyId uint64, err error)
	CancelFly(flyId uint64) error
}

type Notifier interface {
	Notify(data model.FlyPlan) error
}

type plansData struct {
	MaxPlanId      uint64                    `json:"maxPlanId"`
	PlansInfo      map[uint64]*model.FlyPlan `json:"everyDayPlans,omitempty"`
	plansInfoMutex *sync.Mutex
}
type Planer struct {
	notifier       Notifier
	plansData      plansData
	notifyMap      map[uint64]*time.Timer
	notifyMapMutex *sync.Mutex
}

//SetNotifier ...
func (p *Planer) SetNotifier(notifier Notifier) {
	p.notifier = notifier
}

//Start ...
func (p *Planer) Start() error {
	if p.notifier == nil {
		return errors.New("notifier not defined")
	}
	ticker := time.NewTicker(updateNotifyTime)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				p.updateNotificationList()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return nil
}

func (p *Planer) sendNotify(data model.FlyPlan) error {
	flyId := data.FlyId
	if !data.IsEveryDayPlan {
		p.notifyMapMutex.Lock()
		defer p.notifyMapMutex.Unlock()
		delete(p.notifyMap, flyId)
		p.plansData.plansInfoMutex.Lock()
		defer p.plansData.plansInfoMutex.Unlock()
		fmt.Println("Before:", len(p.plansData.PlansInfo[flyId].Notifications), p.plansData.PlansInfo[flyId].Notifications)
		for i, timeDir := range p.plansData.PlansInfo[flyId].Notifications {

			if p.plansData.PlansInfo[flyId].FlyDateTime.Sub(time.Now())+timeDir < 0 {
				p.plansData.PlansInfo[flyId].Notifications[i] =
					p.plansData.PlansInfo[flyId].Notifications[len(p.plansData.PlansInfo[flyId].Notifications)-1]
				p.plansData.PlansInfo[flyId].Notifications = p.plansData.PlansInfo[flyId].Notifications[:len(p.plansData.PlansInfo[flyId].Notifications)-1]
			}

		}
		fmt.Println("After:", len(p.plansData.PlansInfo[flyId].Notifications), p.plansData.PlansInfo[flyId].Notifications)
		if len(p.plansData.PlansInfo[flyId].Notifications) == 0 && p.plansData.PlansInfo[flyId].FlyDateTime.Sub(time.Now()) < 0 {
			delete(p.plansData.PlansInfo, flyId)
		}
	}
	err := p.notifier.Notify(data)

	return err
}

func (p *Planer) updateNotificationList() {
	//timers and plans map lock
	p.notifyMapMutex.Lock()
	defer p.notifyMapMutex.Unlock()
	p.plansData.plansInfoMutex.Lock()
	defer p.plansData.plansInfoMutex.Unlock()

	for _, plan := range p.plansData.PlansInfo {
		if plan.Notifications != nil && len(plan.Notifications) > 0 {
			for _, notification := range plan.Notifications {
				deltaT := plan.FlyDateTime.Sub(time.Now()) + notification
				if plan.IsEveryDayPlan {
					deltaT = deltaT % (time.Hour * 24)
				}
				if deltaT <= updateNotifyTime && deltaT > -2*time.Second {
					fmt.Println("Adding ", deltaT)
					currentPlan := plan
					//if _, ok := p.notifyMap[plan.FlyId]; !ok {
					p.notifyMap[plan.FlyId] = time.AfterFunc(deltaT, func() {
						p.sendNotify(*currentPlan)
						p.notifyMapMutex.Lock()
						defer p.notifyMapMutex.Unlock()
						delete(p.notifyMap, currentPlan.FlyId)
					})
					//}
				}
			}
		}
		deltaT := plan.FlyDateTime.Sub(time.Now())
		if plan.IsEveryDayPlan {
			deltaT = deltaT % (time.Hour * 24)
			if deltaT < 0 {
				deltaT += time.Hour * 24
			}
			fmt.Println("Every day", deltaT)
		}
		if deltaT <= updateNotifyTime && deltaT > -2*time.Second {
			currentPlan := plan
			if _, ok := p.notifyMap[plan.FlyId]; !ok {
				p.notifyMap[plan.FlyId] = time.AfterFunc(deltaT, func() {
					p.sendNotify(*currentPlan)
					p.notifyMapMutex.Lock()
					defer p.notifyMapMutex.Unlock()
					delete(p.notifyMap, currentPlan.FlyId)
				})
			}
		}
	}
}

func (p *Planer) Init() {
	err := p.loadPlans("file.json")
	if err != nil {
		log.Fatal(err)
	}
}

//PlanFly ...
func (p *Planer) PlanFly(info model.FlyPlan) (flyId uint64, err error) {
	defer p.updateNotificationList()
	p.plansData.MaxPlanId++
	info.FlyId = p.plansData.MaxPlanId

	p.plansData.plansInfoMutex.Lock()
	defer p.plansData.plansInfoMutex.Unlock()
	p.plansData.PlansInfo[info.FlyId] = &info
	return info.FlyId, err
}

//CancelFly ...
func (p *Planer) CancelFly(flyId uint64) error {
	p.notifyMapMutex.Lock()
	defer p.notifyMapMutex.Unlock()
	if timer, ok := p.notifyMap[flyId]; ok {
		timer.Stop()
		delete(p.notifyMap, flyId)
	}
	p.notifyMapMutex.Lock()
	defer p.notifyMapMutex.Unlock()
	if timer, ok := p.notifyMap[flyId]; ok {
		timer.Stop()
		delete(p.notifyMap, flyId)
	}
	p.plansData.plansInfoMutex.Lock()
	defer p.plansData.plansInfoMutex.Unlock()
	if _, ok := p.plansData.PlansInfo[flyId]; ok {
		delete(p.plansData.PlansInfo, flyId)
		return nil
	}
	return errors.New("id not exist")
}

//NewPlaner ...
func NewPlaner() *Planer {
	return &Planer{plansData: plansData{
		MaxPlanId:      0,
		PlansInfo:      map[uint64]*model.FlyPlan{},
		plansInfoMutex: &sync.Mutex{},
	},
		notifyMap:      map[uint64]*time.Timer{},
		notifyMapMutex: &sync.Mutex{}}
}

//loadPlans ...
func (p *Planer) loadPlans(filePath string) error {
	var plansData = &plansData{
		MaxPlanId:      0,
		plansInfoMutex: &sync.Mutex{},
		PlansInfo:      map[uint64]*model.FlyPlan{},
	}

	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0644)
	defer file.Close()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(plansData)
	if err != nil {
		if err == io.EOF {
			fmt.Println("Creating new data file")
			return nil
		}
		return err
	}
	p.plansData = *plansData
	p.updateNotificationList()
	return nil
}

//SavePlans ...
func (p Planer) SavePlans(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	decoder := json.NewEncoder(file)
	err = decoder.Encode(p.plansData)
	if err != nil {
		return err
	}
	return nil
}
