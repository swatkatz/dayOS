package graph

type Resolver struct {
	RoutineStore RoutineStore
	TaskStore    TaskStore
	ContextStore ContextStore
	DayPlanStore DayPlanStore
}
