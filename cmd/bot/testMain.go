package main

import (
	"fmt"
	"github.com/japersik/safe-flight-bot/internal/flyPlanner"
	"github.com/japersik/safe-flight-bot/model"
	"time"
)

type NotifierTest struct{}

func (n NotifierTest) Notify(data model.FlyPlan) error {
	fmt.Println("-->>>>", data)
	return nil
}

func main() {
	planner := flyPlanner.NewPlaner()
	planner.SetNotifier(&NotifierTest{})
	planner.Init()
	planner.Start()
	planner.PlanFly(model.FlyPlan{
		Data: model.FlyData{
			Coordinate: model.Coordinate{123, 123},
			Radius:     123,
			UserId:     123,
		},
		FlyId:          0,
		FlyDateTime:    time.Now().Add(5 * time.Second),
		Notifications:  []time.Duration{10 * time.Second, 25 * time.Second, time.Second * 40},
		IsEveryDayPlan: false,
	})
	//fmt.Println(id)
	//fmt.Println(planner)
	time.Sleep(9 * time.Second)
	//planner.CancelFly(id)
	//fmt.Println(planner)

	time.Sleep(1 * time.Minute)
	err := planner.SavePlans("file.json")
	fmt.Println(err)
}
