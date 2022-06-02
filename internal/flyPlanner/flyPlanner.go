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

const updateNotifyTime = 8

type Planner interface {
	SetNotifier(notifier Notifier)
	PlanFly(info model.FlyPlan) (flyId uint64, err error)
	CancelFly(flyId uint64) error
}

type Notifier interface {
	Notify(data model.FlyPlan) error
}

type plansData struct {
	PlanId             uint64          `json:"planId"`
	EveryDayPlans      []model.FlyPlan `json:"everyDayPlans"`
	DateTimePlans      []model.FlyPlan `json:"dateTimePlans"`
	everyDayPlansMutex *sync.Mutex
	dateTimePlansMutex *sync.Mutex
}
type Planer struct {
	notifier       Notifier
	plansData      plansData
	notifyMap      map[uint64]*time.Timer
	notifyMapMutex *sync.Mutex
}

func (p *Planer) SetNotifier(notifier Notifier) {
	p.notifier = notifier
}
func (p *Planer) Start() error {
	if p.notifier == nil {
		return errors.New("notifier not defined")
	}
	ticker := time.NewTicker(updateNotifyTime * time.Second)
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
func (p *Planer) updateNotificationList() {
	dayTimeNow := (time.Now().Hour()*60+time.Now().Minute())*60 + time.Now().Second()
	p.notifyMapMutex.Lock()
	defer p.notifyMapMutex.Unlock()
	p.plansData.dateTimePlansMutex.Lock()
	defer p.plansData.dateTimePlansMutex.Unlock()
	//for i, plan := range p.plansData.DateTimePlans {
	//	dayTime := plan.FlyDateTime.Hour()*60 + plan.FlyDateTime.Minute()
	//	deltaT := dayTime - dayTimeNow
	//	if deltaT < updateNotifyTime {
	//		//добавить в задание
	//	}
	//}
	p.plansData.everyDayPlansMutex.Lock()
	defer p.plansData.everyDayPlansMutex.Unlock()
	for _, plan := range p.plansData.EveryDayPlans {
		dayTime := (plan.FlyDateTime.Hour()*60+plan.FlyDateTime.Minute())*60 + plan.FlyDateTime.Second()
		deltaT := dayTime - dayTimeNow
		if deltaT <= updateNotifyTime && deltaT > -1 {
			fmt.Println(deltaT)
			currentPlan := plan
			fmt.Println("Adding", plan.Data)
			fmt.Println(plan)
			if _, ok := p.notifyMap[plan.FlyId]; !ok {
				p.notifyMap[plan.FlyId] = time.AfterFunc(time.Second*time.Duration(deltaT), func() {
					p.notifier.Notify(currentPlan)
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

func (p *Planer) PlanFly(info model.FlyPlan) (flyId uint64, err error) {
	defer p.updateNotificationList()
	p.plansData.PlanId++
	info.FlyId = p.plansData.PlanId
	if info.IsEveryDayPlan {
		p.plansData.everyDayPlansMutex.Lock()
		defer p.plansData.everyDayPlansMutex.Unlock()
		p.plansData.EveryDayPlans = append(p.plansData.EveryDayPlans, info)
	} else {
		p.plansData.dateTimePlansMutex.Lock()
		defer p.plansData.dateTimePlansMutex.Unlock()
		p.plansData.DateTimePlans = append(p.plansData.DateTimePlans, info)
	}

	return info.FlyId, err
}

func (p *Planer) CancelFly(flyId uint64) error {
	p.notifyMapMutex.Lock()
	defer p.notifyMapMutex.Unlock()
	if timer, ok := p.notifyMap[flyId]; ok {
		timer.Stop()
		delete(p.notifyMap, flyId)
	}

	p.plansData.dateTimePlansMutex.Lock()
	defer p.plansData.dateTimePlansMutex.Unlock()
	for i, plan := range p.plansData.DateTimePlans {
		if plan.FlyId != flyId {
			continue
		}
		p.plansData.DateTimePlans[i] = p.plansData.DateTimePlans[len(p.plansData.DateTimePlans)-1]
		p.plansData.DateTimePlans = p.plansData.DateTimePlans[:len(p.plansData.DateTimePlans)-1]
		return nil
	}
	p.plansData.everyDayPlansMutex.Lock()
	defer p.plansData.everyDayPlansMutex.Unlock()
	for i, plan := range p.plansData.EveryDayPlans {
		if plan.FlyId != flyId {
			continue
		}
		p.plansData.EveryDayPlans[i] = p.plansData.EveryDayPlans[len(p.plansData.EveryDayPlans)-1]
		p.plansData.EveryDayPlans = p.plansData.EveryDayPlans[:len(p.plansData.EveryDayPlans)-1]
		return nil
	}
	return errors.New("id not exist")
}

func NewPlaner() *Planer {
	return &Planer{plansData: plansData{
		PlanId:             0,
		EveryDayPlans:      nil,
		DateTimePlans:      nil,
		everyDayPlansMutex: &sync.Mutex{},
		dateTimePlansMutex: &sync.Mutex{},
	},
		notifyMap:      map[uint64]*time.Timer{},
		notifyMapMutex: &sync.Mutex{}}
}
func (p *Planer) loadPlans(filePath string) error {
	var plansData = &plansData{
		everyDayPlansMutex: &sync.Mutex{},
		dateTimePlansMutex: &sync.Mutex{},
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
	//fmt.Println(plansData)
	return nil
}

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
