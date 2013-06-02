package aam

import (
  "errors"
  "fmt"
  "strconv"
)

type Actor struct {
  reference ActorReference
  behavior Behavior
  behaviorSet bool
  configuration *configuration
  effect Effect
  effectSet bool
}

func (actor *Actor) Signal(error error) {
  actor.effect.errors = append(actor.effect.errors, error)
}

func (actor *Actor) String() string {
  return actor.reference.String()
}

type ActorReference struct {
  id int
  configuration *configuration
}

func (actorReference *ActorReference) Become(behavior Behavior) {
  actor := actorReference.configuration.idToActorMap[actorReference.id]
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
  actor := actorReference.configuration.idToActorMap[actorReference.id]
  event := actor.configuration.Event(target, message)
  actor.effect.events = append(actor.effect.events, event)
}

func (actorReference *ActorReference) Signal(error error) {
  actor := actorReference.configuration.idToActorMap[actorReference.id]
  actor.effect.errors = append(actor.effect.errors, error)
}

func (actorReference *ActorReference) String() string {
  return "{actor: " + strconv.Itoa(actorReference.id) + "}"
}

type Behavior func(event Event)

type Effect struct {
  errors []error
  behavior Behavior
  events []Event
}

type Event struct {
  id EventReference
  configuration *configuration
  target ActorReference
  Message Message
}

func (event *Event) Self() ActorReference {
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
  idToActorMap map[int]*Actor
  nextEventId EventReference
  events []Event
  Trace bool
}

func (configuration *configuration) Create(behavior Behavior) ActorReference {
  actor := Actor{
    reference: ActorReference{id: configuration.nextActorId, configuration: configuration},
    behavior: behavior,
    configuration: configuration}
  configuration.nextActorId++
  configuration.idToActorMap[actor.reference.id] = &actor

  if configuration.Trace {
    fmt.Println("[trace] CREATED    ", actor.String())
  }

  return actor.reference
}

func (configuration *configuration) Dispatch() {
  if len(configuration.events) == 0 {
    configuration.Signal([]error{errors.New("no events to dispatch")})
    return
  }
  event := configuration.events[0]
  configuration.events = configuration.events[1:]
  actor := configuration.idToActorMap[event.target.id]
  if actor.effectSet == false {
    actor.effect = Effect{behavior: actor.behavior}
    actor.effectSet = true

    if configuration.Trace {
      fmt.Println("[trace] DISPATCHING", event.String())
    }

    actor.behavior(event)
    if len(actor.effect.errors) == 0 {
      replacementActor := Actor{
        reference: actor.reference,
        behavior: actor.effect.behavior}
      configuration.idToActorMap[actor.reference.id] = &replacementActor
      for _, _event := range actor.effect.events {
        configuration.events = append(configuration.events, _event)
      }
    } else {
      configuration.Signal(actor.effect.errors)
    }
  } else {
    // TODO: reconsider this implementation, it can starve a particular message
    configuration.events = append(configuration.events, event)
  }

  if configuration.Trace {
    fmt.Println("[trace] DISPATCHED ", event.String())
  }
}

func (configuration *configuration) Event(target ActorReference, message Message) Event {
  event := Event{
    id: configuration.nextEventId,
    configuration: configuration,
    target: target,
    Message: message}

  configuration.nextEventId++

  return event
}

func (configuration *configuration) HasEvents() bool {
  return len(configuration.events) > 0
}

func (configuration *configuration) Send(target ActorReference, message Message) {
  event := configuration.Event(target, message)
  configuration.events = append(configuration.events, event)
}

func (configuration *configuration) Signal(errors []error) {
  fmt.Println("ERRORS:", errors)
}