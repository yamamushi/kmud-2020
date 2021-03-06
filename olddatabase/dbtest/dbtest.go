package dbtest

import (
	"github.com/yamamushi/kmud-2020/color"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/yamamushi/kmud-2020/testutils"
	"github.com/yamamushi/kmud-2020/types"
	"gopkg.in/mgo.v2/bson"
)

type TestSession struct {
}

func (ms TestSession) DB(dbName string) olddatabase.Database {
	return &TestDatabase{}
}

type TestDatabase struct {
}

func (md TestDatabase) C(collectionName string) olddatabase.Collection {
	return &TestCollection{}
}

type TestCollection struct {
}

func (mc TestCollection) Find(selector interface{}) olddatabase.Query {
	return &TestQuery{}
}

func (mc TestCollection) FindId(selector interface{}) olddatabase.Query {
	return &TestQuery{}
}

func (mc TestCollection) RemoveId(id interface{}) error {
	return nil
}

func (mc TestCollection) Remove(selector interface{}) error {
	return nil
}

func (mc TestCollection) DropCollection() error {
	return nil
}

func (mc TestCollection) UpdateId(id interface{}, change interface{}) error {
	return nil
}

func (mc TestCollection) UpsertId(id interface{}, change interface{}) error {
	return nil
}

type TestQuery struct {
}

func (mq TestQuery) Count() (int, error) {
	return 0, nil
}

func (mq TestQuery) One(result interface{}) error {
	return nil
}

func (mq TestQuery) Iter() olddatabase.Iterator {
	return &TestIterator{}
}

type TestIterator struct {
}

func (mi TestIterator) All(result interface{}) error {
	return nil
}

func Test_ThreadSafety(t *testing.T) {
	runtime.GOMAXPROCS(2)
	olddatabase.Init(&TestSession{}, "unit_dbtest")

	char := olddatabase.NewPc("test", testutils.MockId(""), testutils.MockId(""))

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		for i := 0; i < 100; i++ {
			char.SetName(strconv.Itoa(i))
		}
		wg.Done()
	}()

	go func() {
		for i := 0; i < 100; i++ {
		}
		wg.Done()
	}()

	wg.Wait()
}

func Test_User(t *testing.T) {
	user := olddatabase.NewUser("testuser", "", false)

	if user.IsOnline() {
		t.Errorf("Newly created user shouldn't be online")
	}

	user.SetOnline(true)

	testutils.Assert(user.IsOnline(), t, "Call to SetOnline(true) failed")
	testutils.Assert(user.GetColorMode() == color.ModeNone, t, "Newly created user should have a color mode of None")

	user.SetColorMode(color.ModeLight)

	testutils.Assert(user.GetColorMode() == color.ModeLight, t, "Call to SetColorMode(types.ModeLight) failed")

	user.SetColorMode(color.ModeDark)

	testutils.Assert(user.GetColorMode() == color.ModeDark, t, "Call to SetColorMode(types.ModeDark) failed")

	pw := "password"
	user.SetPassword(pw)

	testutils.Assert(user.VerifyPassword(pw), t, "User password verification failed")

	width := 11
	height := 12
	user.SetWindowSize(width, height)

	testWidth, testHeight := user.GetWindowSize()

	testutils.Assert(testWidth == width && testHeight == height, t, "Call to SetWindowSize() failed")

	terminalType := "fake terminal type"
	user.SetTerminalType(terminalType)

	testutils.Assert(terminalType == user.GetTerminalType(), t, "Call to SetTerminalType() failed")
}

func Test_PlayerCharacter(t *testing.T) {
	fakeId := bson.ObjectId("12345")
	character := olddatabase.NewPc("testcharacter", fakeId, fakeId)

	testutils.Assert(character.GetUserId() == fakeId, t, "Call to character.SetUser() failed", fakeId, character.GetUserId())
	testutils.Assert(!character.IsOnline(), t, "Player-Characters should be offline by default")

	character.SetOnline(true)

	testutils.Assert(character.IsOnline(), t, "Call to character.SetOnline(true) failed")

	character.SetRoomId(fakeId)

	testutils.Assert(character.GetRoomId() == fakeId, t, "Call to character.SetRoom() failed", fakeId, character.GetRoomId())

	cashAmount := 1234
	character.SetCash(cashAmount)

	testutils.Assert(character.GetCash() == cashAmount, t, "Call to character.GetCash() failed", cashAmount, character.GetCash())

	character.AddCash(cashAmount)

	testutils.Assert(character.GetCash() == cashAmount*2, t, "Call to character.AddCash() failed", cashAmount*2, character.GetCash())

	// conversation := "this is a fake conversation that is made up for the unit test"

	// character.SetConversation(conversation)

	// testutils.Assert(character.GetConversation() == conversation, t, "Call to character.SetConversation() failed")

	health := 123

	character.SetHealth(health)

	testutils.Assert(character.GetHealth() == health, t, "Call to character.SetHealth() failed")

	hitpoints := health - 10

	character.SetHitPoints(hitpoints)

	testutils.Assert(character.GetHitPoints() == hitpoints, t, "Call to character.SetHitPoints() failed")

	character.SetHitPoints(health + 10)

	testutils.Assert(character.GetHitPoints() == health, t, "Shouldn't be able to set a character's hitpoints to be greater than its maximum health", health, character.GetHitPoints())

	character.SetHealth(character.GetHealth() - 10)

	testutils.Assert(character.GetHitPoints() == character.GetHealth(), t, "Lowering health didn't lower the hitpoint count along with it", character.GetHitPoints(), character.GetHealth())

	character.SetHealth(100)
	character.SetHitPoints(100)

	hitAmount := 51
	character.Hit(hitAmount)

	testutils.Assert(character.GetHitPoints() == character.GetHealth()-hitAmount, t, "Call to character.Hit() failed", hitAmount, character.GetHitPoints())

	character.Heal(hitAmount)

	testutils.Assert(character.GetHitPoints() == character.GetHealth(), t, "Call to character.Heal() failed", hitAmount, character.GetHitPoints())
}

func Test_Zone(t *testing.T) {
	zoneName := "testzone"
	zone := olddatabase.NewZone(zoneName)

	testutils.Assert(zone.GetName() == zoneName, t, "Zone didn't have correct name upon creation", zoneName, zone.GetName())
}

func Test_Room(t *testing.T) {
	fakeZoneId := bson.ObjectId("!2345")
	room := olddatabase.NewRoom(fakeZoneId, types.Coordinate{X: 0, Y: 0, Z: 0})

	testutils.Assert(room.GetZoneId() == fakeZoneId, t, "Room didn't have correct zone ID upon creation", fakeZoneId, room.GetZoneId())

	fakeZoneId2 := bson.ObjectId("11111")
	room.SetZoneId(fakeZoneId2)
	testutils.Assert(room.GetZoneId() == fakeZoneId2, t, "Call to room.SetZoneId() failed")

	directionList := make([]types.Direction, 10)
	directionCount := 10

	for i := 0; i < directionCount; i++ {
		directionList[i] = types.Direction(i)
	}

	for _, dir := range directionList {
		testutils.Assert(!room.HasExit(dir), t, "Room shouldn't have any exits enabled by default", dir)
		room.SetExitEnabled(dir, true)
		testutils.Assert(room.HasExit(dir), t, "Call to room.SetExitEnabled(true) failed")
		room.SetExitEnabled(dir, false)
		testutils.Assert(!room.HasExit(dir), t, "Call to room.SetExitEnabled(false) failed")
	}

	title := "Test ServerName"
	room.SetTitle(title)
	testutils.Assert(title == room.GetTitle(), t, "Call to room.SetTitle() failed", title, room.GetTitle())

	description := "This is a fake description"
	room.SetDescription(description)
	testutils.Assert(description == room.GetDescription(), t, "Call to room.SetDescription() failed", description, room.GetDescription())

	coord := types.Coordinate{X: 1, Y: 2, Z: 3}
	room.SetLocation(coord)
	testutils.Assert(coord == room.GetLocation(), t, "Call to room.SetLocation() failed", coord, room.GetLocation())
}

func Test_Item(t *testing.T) {
	name := "test_item"
	template := olddatabase.NewTemplate(name)
	item := olddatabase.NewItem(template.GetId())

	testutils.Assert(item.GetName() == name, t, "Item didn't get created with correct name", name, item.GetName())
}
