// -*- Mote: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2016 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package interfaces_test

import (
	. "gopkg.in/check.v1"

	. "github.com/ubuntu-core/snappy/interfaces"
	"github.com/ubuntu-core/snappy/snap"
	"github.com/ubuntu-core/snappy/testutil"
)

type SecuritySuite struct {
	testutil.BaseTest
	repo *Repository
	plug *Plug
	slot *Slot
}

var _ = Suite(&SecuritySuite{})

func (s *SecuritySuite) SetUpTest(c *C) {
	s.BaseTest.SetUpTest(c)
	s.repo = NewRepository()
	// NOTE: the names producer/consumer are confusing. They will be fixed shortly.
	producer, err := snap.InfoFromSnapYaml([]byte(`
name: producer
apps:
    hook:
plugs:
    plug: interface
`))
	c.Assert(err, IsNil)
	consumer, err := snap.InfoFromSnapYaml([]byte(`
name: consumer
apps:
    app:
slots:
    slot:
        interface: interface
        label: label
        attr: value
`))
	c.Assert(err, IsNil)
	s.plug = &Plug{PlugInfo: producer.Plugs["plug"]}
	s.slot = &Slot{SlotInfo: consumer.Slots["slot"]}
	// TODO: make this obsolete thanks to unified and rich snap.Info
	MockActiveSnapMetaData(&s.BaseTest, func(snapName string) (string, string, []string, error) {
		switch snapName {
		case "producer":
			return "version", "origin", []string{"hook"}, nil
		case "consumer":
			return "version", "origin", []string{"app"}, nil
		default:
			panic("unexpected snap name")
		}
	})
}

func (s *SecuritySuite) prepareFixtureWithInterface(c *C, i Interface) {
	err := s.repo.AddInterface(i)
	c.Assert(err, IsNil)
	err = s.repo.AddPlug(s.plug)
	c.Assert(err, IsNil)
	err = s.repo.AddSlot(s.slot)
	c.Assert(err, IsNil)
	err = s.repo.Connect(s.plug.Snap.Name, s.plug.Name, s.slot.Snap.Name, s.slot.Name)
	c.Assert(err, IsNil)
}

// Tests for uDev

func (s *SecuritySuite) TestUdevPlugPermissions(c *C) {
	s.prepareFixtureWithInterface(c, &TestInterface{
		InterfaceName: "interface",
		PlugSnippetCallback: func(plug *Plug, slot *Slot, securitySystem SecuritySystem) ([]byte, error) {
			if securitySystem == SecurityUDev {
				return []byte("...\n"), nil
			}
			return nil, nil
		},
	})
	// Ensure that plug-side security profile looks correct.
	blobs, err := s.repo.SecurityFilesForSnap(s.plug.Snap.Name)
	c.Assert(err, IsNil)
	c.Check(blobs["/etc/udev/rules.d/70-producer.hook.snap.rules"], DeepEquals, []byte("...\n"))
}

func (s *SecuritySuite) TestUdevSlotPermissions(c *C) {
	s.prepareFixtureWithInterface(c, &TestInterface{
		InterfaceName: "interface",
		SlotSnippetCallback: func(plug *Plug, slot *Slot, securitySystem SecuritySystem) ([]byte, error) {
			if securitySystem == SecurityUDev {
				return []byte("...\n"), nil
			}
			return nil, nil
		},
	})
	// Ensure that slot-side security profile looks correct.
	blobs, err := s.repo.SecurityFilesForSnap(s.slot.Snap.Name)
	c.Assert(err, IsNil)
	c.Check(blobs["/etc/udev/rules.d/70-consumer.app.snap.rules"], DeepEquals, []byte("...\n"))
}