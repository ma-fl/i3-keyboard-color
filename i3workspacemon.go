package main

import (
  "log"
  "fmt"
  "encoding/json"
  "go.i3wm.org/i3"
  zmq "github.com/pebbe/zmq4"
)

const IPC_PATH = "/tmp/duckycolord"

type color struct {
  empty bool
  r,g,b byte
}

// colors for worksace states, should be configurable
var colorExisting = color{false, 0x00, 0xff, 0xff}
var colorNotExisting = color{true, 0x00, 0x00, 0x00}
var colorActive = color{false, 0xaa, 0xaa, 0xff}
var colorUrgent = color{false, 0xff, 0xcc, 0x00}

// map of workspace number to key name
// this should be configurable and should match the i3 config
var keymap = map[int64]string {
  1: "1",
  2: "2",
  3: "3",
  4: "4",
  5: "5",
  6: "6",
  7: "7",
  8: "8",
  9: "9",
  10: "0",
  11: "F1",
  12: "F2",
  13: "F3",
  14: "F4",
  15: "F5",
  16: "F6",
  17: "F7",
  18: "F8",
  19: "F9",
  20: "F10"}


func getJSON(keys map[string]color) ([]byte, error) {
  plain := make(map[string]interface{})
  for key, col := range keys {
    if col.empty {
      plain[key] = false
    } else {
      colorlist := [3]byte{col.r, col.g, col.b}
      plain[key] = colorlist
    }
  }
  return json.Marshal(plain)
}

func updateWorkspaces(keys map[string]color, colord *zmq.Socket)() {
  workspaces, err := i3.GetWorkspaces()
  if err != nil {
    log.Fatal(err)
  }
  existing_workspaces := make(map[int64]int)
  anythingUrgent := false
  // look at each workspace and color the corresponding key
  for _, ws := range workspaces {
    if _, ok := keymap[ws.Num]; ok {
      existing_workspaces[ws.Num] = 1
      if ws.Urgent {
        keys[keymap[ws.Num]] = colorUrgent
        anythingUrgent = true
      } else if ws.Focused {
        keys[keymap[ws.Num]] = colorActive
      } else {
        keys[keymap[ws.Num]] = colorExisting
      }
    }
    fmt.Printf("[%d] (%s)\n", ws.Num, ws.Name)
    fmt.Printf("Urgent: %t\n", ws.Urgent)
  }
  // find workspaces which we did not process == are not there right now
  for num, key := range keymap {
    if _, ok := existing_workspaces[num]; !ok {
      keys[key] = colorNotExisting
    }
  }
  // change color of space, if anything is urgent
  if anythingUrgent {
    keys["space"] = colorUrgent
  } else {
    keys["space"] = colorNotExisting
  }
  fmt.Printf("%v\n", keys)
  msg, err := getJSON(keys)
  fmt.Printf("%v\n", msg)
  if _, err := colord.SendBytes(msg, 0); err != nil {
    log.Fatal(err)
  }
}


func main()() {

  // map of keys with colors for all possible workspaces
  keyboard := make(map[string]color)
  for _, key := range keymap {
    keyboard[key] = colorNotExisting
  }

  // connect to ducky color daemon
  colord, err := zmq.NewSocket(zmq.PUSH)
  if err != nil {
    log.Fatal(err)
  }
  defer colord.Close()
  url := fmt.Sprintf("ipc://%s", IPC_PATH)
  if err := colord.Connect(url); err != nil {
    log.Fatal(err)
  }

  // subscribe to i3 window manager desktop events
  recv := i3.Subscribe(i3.WorkspaceEventType)

  // main loop handling i3 events
  for recv.Next() {
    ev := recv.Event().(*i3.WorkspaceEvent)
    log.Printf("change: %s", ev.Change)
    //log.Printf("num: %d (%s), urgent: %t", ev.Current.ID, ev.Current.Name, ev.Current.Urgent)
    updateWorkspaces(keyboard, colord)
  }
  log.Fatal(recv.Close())
}
