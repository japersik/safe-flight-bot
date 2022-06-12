package flyPlanner

import (
	"encoding/json"
	"errors"
	"github.com/japersik/safe-flight-bot/model"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const updateNotifyTime = 120 * time.Second

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
	plansData      *plansData
	notifyMap      map[runningPlan]*time.Timer
	notifyMapMutex *sync.Mutex
	filepath       string
}
type runningPlan struct {
	flyId        uint64
	notification time.Duration
}

//NewPlaner ...
func NewPlaner(filepath string) *Planer {
	return &Planer{plansData: &plansData{
		MaxPlanId:      0,
		PlansInfo:      map[uint64]*model.FlyPlan{},
		plansInfoMutex: &sync.Mutex{},
	},
		notifyMap:      map[runningPlan]*time.Timer{},
		notifyMapMutex: &sync.Mutex{},
		filepath:       filepath}

}

//SetNotifier ...
func (p *Planer) SetNotifier(notifier Notifier) {
	p.notifier = notifier
}

//Start ...
func (p *Planer) Start() error {
	log.Println("notification starting")
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

func (p *Planer) sendNotify(notificationInfo runningPlan) error {
	flyId := notificationInfo.flyId
	p.plansData.plansInfoMutex.Lock()
	defer p.plansData.plansInfoMutex.Unlock()
	if !p.plansData.PlansInfo[flyId].IsEveryDayPlan {
		for i, timeDir := range p.plansData.PlansInfo[flyId].Notifications {
			if i >= len(p.plansData.PlansInfo[flyId].Notifications) {
				break
			}
			if timeDir == notificationInfo.notification {
				p.plansData.PlansInfo[flyId].Notifications[i] =
					p.plansData.PlansInfo[flyId].Notifications[len(p.plansData.PlansInfo[flyId].Notifications)-1]
				p.plansData.PlansInfo[flyId].Notifications = p.plansData.PlansInfo[flyId].Notifications[:len(p.plansData.PlansInfo[flyId].Notifications)-1]
			}
		}
	}
	toSend := *p.plansData.PlansInfo[flyId]
	if !p.plansData.PlansInfo[flyId].IsEveryDayPlan && len(p.plansData.PlansInfo[flyId].Notifications) == 0 &&
		p.plansData.PlansInfo[flyId].FlyDateTime.Sub(time.Now()) < 0 {
		delete(p.plansData.PlansInfo, flyId)
	}
	return p.notifier.Notify(toSend)
}

func (p *Planer) updateNotificationList() {
	//timers and plans map lock

	p.plansData.plansInfoMutex.Lock()
	defer p.plansData.plansInfoMutex.Unlock()
	for _, plan := range p.plansData.PlansInfo {
		if plan.Notifications != nil && len(plan.Notifications) > 0 {
			p.addAllNotifications(*plan)
		}
	}
}

func (p *Planer) addAllNotifications(plan model.FlyPlan) {
	p.notifyMapMutex.Lock()
	defer p.notifyMapMutex.Unlock()
	for _, notification := range plan.Notifications {
		deltaT := plan.FlyDateTime.Sub(time.Now()) + notification
		if plan.IsEveryDayPlan {
			deltaT = deltaT % (time.Hour * 24)
			if deltaT < 0 {
				deltaT += time.Hour * 24
			}
		}
		if deltaT <= updateNotifyTime && deltaT > -updateNotifyTime/2 {
			//fmt.Println("Adding ", deltaT, notification)
			notificationInfo := runningPlan{plan.FlyId, notification}
			if _, ok := p.notifyMap[notificationInfo]; !ok {
				p.notifyMap[notificationInfo] = time.AfterFunc(deltaT, func() {
					p.sendNotify(notificationInfo)
					p.notifyMapMutex.Lock()
					defer p.notifyMapMutex.Unlock()
					delete(p.notifyMap, notificationInfo)
				})
			}
		}
	}
}
func (p *Planer) Init() {
	err := p.loadPlans()
	if err != nil {
		log.Fatal(err)
	}
}

//PlanFly ...
func (p *Planer) PlanFly(info model.FlyPlan) (uint64, error) {
	p.plansData.MaxPlanId++
	info.FlyId = p.plansData.MaxPlanId
	if info.IsEveryDayPlan {
		info.FlyDateTime = info.FlyDateTime.AddDate(time.Now().Year()-info.FlyDateTime.Year(), 0, 0)
	}
	p.plansData.plansInfoMutex.Lock()
	defer p.plansData.plansInfoMutex.Unlock()
	p.plansData.PlansInfo[info.FlyId] = &info
	p.addAllNotifications(info)
	log.Printf("flight No.%d from user %d created at (%f, %f)\n", info.FlyId, info.Data.UserId,
		info.Data.Coordinate.Lat, info.Data.Coordinate.Lng)
	return info.FlyId, nil
}

//CancelFly ...
func (p *Planer) CancelFly(flyId uint64) error {
	log.Printf("flight No.%d canceled\n", flyId)
	p.notifyMapMutex.Lock()
	defer p.notifyMapMutex.Unlock()
	for plan, timer := range p.notifyMap {
		if plan.flyId == flyId {
			timer.Stop()
			delete(p.notifyMap, plan)
		}
	}

	p.plansData.plansInfoMutex.Lock()
	defer p.plansData.plansInfoMutex.Unlock()
	if _, ok := p.plansData.PlansInfo[flyId]; ok {
		delete(p.plansData.PlansInfo, flyId)
		return nil
	}
	return errors.New("id not exist")
}

//loadPlans ...
func (p *Planer) loadPlans() error {
	var plansData = &plansData{
		MaxPlanId:      0,
		plansInfoMutex: &sync.Mutex{},
		PlansInfo:      map[uint64]*model.FlyPlan{},
	}
	file, err := os.OpenFile(p.filepath, os.O_RDONLY|os.O_CREATE, 0644)
	defer file.Close()
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(plansData)
	if err != nil {
		if err == io.EOF {
			log.Println("Creating new data file")
			return nil
		}
		return err
	}
	p.plansData = plansData
	log.Printf("flight plan file loaded. There are %d plans in total. Max id %d", len(plansData.PlansInfo), plansData.MaxPlanId)
	p.updateNotificationList()
	return nil
}

//SavePlans ...
func (p Planer) SavePlans() error {
	file, err := os.OpenFile(p.filepath, os.O_CREATE|os.O_WRONLY, 0644)
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
