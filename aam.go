package aam

import (
  "errors"
  "fmt"
  "strconv"
  "sync"
)

type Actor struct {
  reference ActorReference
  behavior Behavior
  behaviorSet bool
  behaviorSetMutex sync.Mutex
  configuration *configuration
  effect Effect
  effectSet bool
  effectSetMutex sync.Mutex
}

func (actor *Actor) Signal(error error) {
  actor.effect.errorsMutex.Lock()
  actor.effect.errors = append(actor.effect.errors, error)
  actor.effect.errorsMutex.Unlock()
}

func (actor *Actor) String() string {
  return actor.reference.String()
}

type ActorReference struct {
  id int
  configuration *configuration
}

func (actorReference *ActorReference) Become(behavior Behavior) {
  actorReference.configuration.idToActorMapMutex.Lock()
  actor := actorReference.configuration.idToActorMap[actorReference.id]
  actorReference.configuration.idToActorMapMutex.Unlock()
  actor.behaviorSetMutex.Lock()
  defer actor.behaviorSetMutex.Unlock()
  if actor.behaviorSet {
    actor.Signal(errors.New("Behavior can be set once only during message handling"))
  } else {
    actor.effect.behavior = behavior
    actor.behaviorSet = true
  }
}

func (actorReference *ActorReference) Create(behavior Behavior) ActorReference {
  return actorReference.configuration.Create(behavior)
}

func (actorReference *ActorReference) Send(target ActorReference, message Message) {
  actorReference.configuration.idToActorMapMutex.Lock()
  actor := actorReference.configuration.idToActorMap[actorReference.id]
  actorReference.configuration.idToActorMapMutex.Unlock()
  event := actor.configuration.Event(target, message)
  actor.effect.eventsMutex.Lock()
  actor.effect.events = append(actor.effect.events, event)
  actor.effect.eventsMutex.Unlock()
}

func (actorReference *ActorReference) Signal(error error) {
  actorReference.configuration.idToActorMapMutex.Lock()
  actor := actorReference.configuration.idToActorMap[actorReference.id]
  actorReference.configuration.idToActorMapMutex.Unlock()
  actor.effect.errorsMutex.Lock()
  actor.effect.errors = append(actor.effect.errors, error)
  actor.effect.errorsMutex.Unlock()
}

func (actorReference *ActorReference) String() string {
  return "{actor: " + strconv.Itoa(actorReference.id) + "}"
}

type Behavior func(event Event)

type Effect struct {
  errors []error
  errorsMutex sync.Mutex
  behavior Behavior
  events []Event
  eventsMutex sync.Mutex
}

type Event struct {
  id EventReference
  configuration *configuration
  target ActorReference
  Message Message
}

func (event *Event) Self() ActorReference {
  event.configuration.idToActorMapMutex.Lock()
  defer event.configuration.idToActorMapMutex.Unlock()
  return event.configuration.idToActorMap[event.target.id].reference
}

func (event *Event) String() string {
  return "{event: " + event.id.String() + ", target: " + event.target.String() +
    ", message: " + event.Message.String() + "}"
}

type EventReference int

func (eventReference EventReference) String() string {
  return strconv.Itoa(int(eventReference))
}

type Message []interface{}

func (message Message) String() string {
  stringRepresentation := "[ "
  for _, arg := range message {
    stringRepresentation += fmt.Sprint(arg) + " "
  }
  stringRepresentation += "]"
  return stringRepresentation
}

func New() configuration {
  return configuration{
    nextActorId: 1, 
    idToActorMap: make(map[int]*Actor),
    nextEventId: 1}
}

type configuration struct {
  nextActorId int
  nextActorIdMutex sync.Mutex
  idToActorMap map[int]*Actor
  idToActorMapMutex sync.Mutex
  nextEventId EventReference
  nextEventIdMutex sync.Mutex
  events []Event
  eventsMutex sync.Mutex
  Trace bool
}

func (configuration *configuration) Create(behavior Behavior) ActorReference {
  configuration.nextActorIdMutex.Lock()
  defer configuration.nextActorIdMutex.Unlock()

  actor := Actor{
    reference: ActorReference{id: configuration.nextActorId, configuration: configuration},
    behavior: behavior,
    configuration: configuration}
  configuration.nextActorId++

  configuration.idToActorMapMutex.Lock()
  configuration.idToActorMap[actor.reference.id] = &actor
  configuration.idToActorMapMutex.Unlock()

  if configuration.Trace {
    fmt.Println("[trace] CREATED    ", actor.String())
  }

  return actor.reference
}

func (configuration *configuration) Dispatch() {
  configuration.eventsMutex.Lock()
  if len(configuration.events) == 0 {
    configuration.Signal([]error{errors.New("no events to dispatch")})
    configuration.eventsMutex.Unlock()
    return
  }
  event := configuration.events[0]
  configuration.events = configuration.events[1:]
  configuration.eventsMutex.Unlock()

  configuration.idToActorMapMutex.Lock()
  actor := configuration.idToActorMap[event.target.id]
  configuration.idToActorMapMutex.Unlock()

  actor.effectSetMutex.Lock()
  if actor.effectSet == false {
    actor.effect = Effect{behavior: actor.behavior}
    actor.effectSet = true
    actor.effectSetMutex.Unlock()

    if configuration.Trace {
      fmt.Println("[trace] DISPATCHING", event.String())
    }

    actor.behavior(event)
    if len(actor.effect.errors) == 0 {
      replacementActor := Actor{
        reference: actor.reference,
        behavior: actor.effect.behavior}
      configuration.idToActorMapMutex.Lock()
      configuration.idToActorMap[actor.reference.id] = &replacementActor
      configuration.idToActorMapMutex.Unlock()
      configuration.eventsMutex.Lock()
      for _, _event := range actor.effect.events {
        configuration.events = append(configuration.events, _event)
      }
      configuration.eventsMutex.Unlock()
    } else {
      configuration.Signal(actor.effect.errors)
    }
  } else {
    // TODO: reconsider this implementation, it can starve a particular message
    actor.effectSetMutex.Unlock()
    configuration.eventsMutex.Lock()
    configuration.events = append(configuration.events, event)
    configuration.eventsMutex.Unlock()
  }

  if configuration.Trace {
    fmt.Println("[trace] DISPATCHED ", event.String())
  }
}

func (configuration *configuration) Event(target ActorReference, message Message) Event {
  configuration.nextEventIdMutex.Lock()
  defer configuration.nextEventIdMutex.Unlock()

  event := Event{
    id: configuration.nextEventId,
    configuration: configuration,
    target: target,
    Message: message}

  configuration.nextEventId++

  return event
}

func (configuration *configuration) HasEvents() bool {
  configuration.eventsMutex.Lock()
  defer configuration.eventsMutex.Unlock()
  return len(configuration.events) > 0
}

func (configuration *configuration) Send(target ActorReference, message Message) {
  event := configuration.Event(target, message)

  configuration.eventsMutex.Lock()
  configuration.events = append(configuration.events, event)
  configuration.eventsMutex.Unlock()
}

func (configuration *configuration) Signal(errors []error) {
  fmt.Println("ERRORS:", errors)
}