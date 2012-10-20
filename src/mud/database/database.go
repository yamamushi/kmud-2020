package database

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"mud/utils"
)

type dbError struct {
	message string
}

func (self dbError) Error() string {
	return self.message
}

func newDbError(message string) dbError {
	var err dbError
	err.message = message
	return err
}

type collectionName string

func getCollection(session *mgo.Session, collection collectionName) *mgo.Collection {
	return session.DB("mud").C(string(collection))
}

// Collection names
const (
	cUsers      = collectionName("users")
	cCharacters = collectionName("characters")
	cRooms      = collectionName("rooms")
)

// Field names
const (
	fId          = "_id"
	fName        = "name"
	fCharacters  = "characters"
	fRoom        = "room"
	fLocation    = "location"
	fTitle       = "title"
	fDescription = "description"
	fNorth       = "exit_north"
	fEast        = "exit_east"
	fSouth       = "exit_south"
	fWest        = "exit_west"
	fUp          = "exit_up"
	fDown        = "exit_down"
	fDefault     = "default"
)

// DB commands
const (
	SET  = "$set"
	PUSH = "$push"
	PULL = "$pull"
)

func FindUser(session *mgo.Session, name string) (bool, error) {
	c := getCollection(session, cUsers)
	q := c.Find(bson.M{fName: name})

	count, err := q.Count()

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func NewUser(session *mgo.Session, name string) error {

	found, err := FindUser(session, name)

	if err != nil {
		return err
	}

	if found {
		return newDbError("That user already exists")
	}

	c := getCollection(session, cUsers)
	c.Insert(bson.M{fName: name})

	return nil
}

func GetCharacterRoom(session *mgo.Session, character Character) (Room, error) {
	return GetRoom(session, character.RoomId)
}

func FindRoom(session *mgo.Session, query interface{}) (Room, error) {
	c := getCollection(session, cRooms)
	q := c.Find(query)

	count, err := q.Count()

	var room Room
	if err != nil {
		return room, err
	}

	if count == 0 {
		return room, newDbError(fmt.Sprintf("Room not found, query: %v", query))
	}

	err = q.One(&room)

	return room, err
}

func FindCharacter(session *mgo.Session, query interface{}) (Character, error) {
	c := getCollection(session, cCharacters)
	q := c.Find(query)

	count, err := q.Count()

	var character Character
	if err != nil {
		return character, err
	}

	if count == 0 {
		return character, newDbError(fmt.Sprintf("Character not found, query: %v", query))
	}

	err = q.One(&character)

	return character, err
}

func GetCharacter(session *mgo.Session, id bson.ObjectId) (Character, error) {
	return FindCharacter(session, bson.M{fId: id})
}

func GetCharacterByName(session *mgo.Session, name string) (Character, error) {
	return FindCharacter(session, bson.M{fName: name})
}

func GetRoom(session *mgo.Session, id bson.ObjectId) (Room, error) {
	return FindRoom(session, bson.M{fId: id})
}

func GetRoomByLocation(session *mgo.Session, location Coordinate) (Room, error) {
	return FindRoom(session, bson.M{fLocation: location})
}

func CreateRoom(session *mgo.Session, room Room) (Room, error) {
	c := getCollection(session, cRooms)
	err := c.Insert(room)

	if err != nil {
		fmt.Printf("Error creating room: %v\n", err)
		return room, err
	}

	room, err = GetRoomByLocation(session, room.Location)

	return room, err
}

func CreateCharacter(session *mgo.Session, userName string, characterName string) (Character, error) {
	character, err := GetCharacterByName(session, characterName)

	if err == nil {
		return character, newDbError("That character already exists")
	}

	character = newCharacter(characterName)

	characterCollection := getCollection(session, cCharacters)
	err = characterCollection.Insert(character)

	if err != nil {
		fmt.Printf("Error inserting new character object into database: %v\n", err)
		return character, err
	}

	character, err = GetCharacterByName(session, character.Name)

	startingRoom, err := StartingRoom(session)

	if err != nil {
		fmt.Printf("Error getting starting room: %v\n", err)
		return character, err
	}

	character.RoomId = startingRoom.Id
	err = CommitCharacter(session, character)

	if err != nil {
		fmt.Printf("Error committing character object: %v\n", err)
		return character, err
	}

	if err == nil {
		userCollection := getCollection(session, cUsers)
		err = userCollection.Update(bson.M{fName: userName}, bson.M{PUSH: bson.M{fCharacters: character.Id}})

		if err != nil {
			fmt.Printf("Error updating user with new character data: %v\n", err)
		}
	}

	return character, err
}

func SetCharacterRoom(session *mgo.Session, character string, roomId bson.ObjectId) error {
	c := getCollection(session, cCharacters)
	err := c.Update(bson.M{fName: character}, bson.M{SET: bson.M{fRoom: roomId}})

	if err != nil {
		fmt.Printf("Failed setting character room :%v\n", err)
	}

	return err
}

func GetUserCharacters(session *mgo.Session, userName string) ([]Character, error) {
	c := getCollection(session, cUsers)
	q := c.Find(bson.M{fName: userName})

	result := map[string][]bson.ObjectId{}
	err := q.One(&result)

	var characters []Character
	for _, charId := range result[fCharacters] {
		character, err := GetCharacter(session, charId)

		if err != nil {
			fmt.Printf("Failed to find character with id %s, belonging to user %s: %s\n", charId, userName, err)
		} else {
			characters = append(characters, character)
		}
	}

	return characters, err
}

func DeleteCharacter(session *mgo.Session, user string, character string) error {
	c := getCollection(session, cUsers)
	c.Update(bson.M{fName: user}, bson.M{PULL: bson.M{fCharacters: utils.Simplify(character)}})

	c = getCollection(session, cCharacters)
	c.Remove(bson.M{fName: character})

	return nil
}

func StartingRoom(session *mgo.Session) (Room, error) {
	c := getCollection(session, cRooms)
	q := c.Find(bson.M{fDefault: true})

	count, err := q.Count()

	var room Room
	if err != nil {
		return room, err
	}

	if count == 0 {
		return room, newDbError("No default room found")
	}

	if count > 1 {
		fmt.Printf("Warning: More than one default room found\n")
	}

	err = q.One(&room)

	return room, err
}

func GenerateDefaultMap(session *mgo.Session) {
	c := getCollection(session, cRooms)
	c.DropCollection()

	room := newRoom()
	room.Location = Coordinate{0, 0, 0}
	room.Default = true

	CreateRoom(session, room)
}

func SetRoomTitle(session *mgo.Session, room Room, title string) error {
	c := getCollection(session, cRooms)
	return c.Update(bson.M{fId: room.Id}, bson.M{SET: bson.M{fTitle: title}})
}

func SetRoomDescription(session *mgo.Session, room Room, description string) error {
	c := getCollection(session, cRooms)
	return c.Update(bson.M{fId: room.Id}, bson.M{SET: bson.M{fDescription: description}})
}

func directionToFieldName(direction ExitDirection) string {
	switch direction {
	case DirectionNorth:
		return fNorth
	case DirectionEast:
		return fEast
	case DirectionSouth:
		return fSouth
	case DirectionWest:
		return fWest
	case DirectionUp:
		return fUp
	case DirectionDown:
		return fDown
	}

	// Wouldn't ever expect DirectionNone to be passed here
	panic("Unexpected code path")
}

func CommitRoom(session *mgo.Session, room Room) error {
	c := getCollection(session, cRooms)
	return c.Update(bson.M{fId: room.Id}, room)
}

func CommitCharacter(session *mgo.Session, character Character) error {
	c := getCollection(session, cCharacters)
	return c.Update(bson.M{fId: character.Id}, character)
}

func MoveCharacter(session *mgo.Session, character *Character, direction ExitDirection) (Room, error) {
	room, err := GetRoom(session, character.RoomId)

	if err != nil {
		return room, err
	}

	newLocation := room.Location

	switch direction {
	case DirectionNorth:
		newLocation.Y -= 1
	case DirectionEast:
		newLocation.X += 1
	case DirectionSouth:
		newLocation.Y += 1
	case DirectionWest:
		newLocation.X -= 1
	case DirectionUp:
		newLocation.Z -= 1
	case DirectionDown:
		newLocation.Z += 1
	default:
		panic("Unexpected code path")
	}

	room, err = GetRoomByLocation(session, newLocation)

	if err != nil {
		fmt.Printf("No room found at location %v, creating a new one\n", newLocation)
		room = newRoom()

		switch direction {
		case DirectionNorth:
			room.ExitSouth = true
		case DirectionEast:
			room.ExitWest = true
		case DirectionSouth:
			room.ExitNorth = true
		case DirectionWest:
			room.ExitEast = true
		case DirectionUp:
			room.ExitDown = true
		case DirectionDown:
			room.ExitUp = true
		default:
			panic("Unexpected code path")
		}

		room.Location = newLocation
		room, err = CreateRoom(session, room)
	} else {
		character.RoomId = room.Id
		err = CommitCharacter(session, *character)
	}

	return room, err
}

// vim: nocindent
