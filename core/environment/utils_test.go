/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
 * Author: Piotr Konopka <piotr.konopka@cern.ch>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * In applying this license CERN does not waive the privileges and
 * immunities granted to it by virtue of its status as an
 * Intergovernmental Organization or submit itself to any jurisdiction.
 */

package environment

import (
	"github.com/looplab/fsm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HandleFailedGoError", func() {
	It("does not overwrite state for InvalidEventError", func() {
		env := &Environment{}
		env.Sm = fsm.NewFSM("DONE", fsm.Events{}, fsm.Callbacks{})
		Expect(env.Sm.Current()).To(Equal("DONE"))

		HandleFailedGoError(fsm.InvalidEventError{Event: "GO_ERROR", State: "DONE"}, env)
		Expect(env.Sm.Current()).To(Equal("DONE"))
	})

	It("overwrites state to ERROR for other errors", func() {
		env := &Environment{}
		env.Sm = fsm.NewFSM("CONFIGURED", fsm.Events{}, fsm.Callbacks{})

		HandleFailedGoError(fsm.UnknownEventError{Event: "BOOM"}, env)
		Expect(env.Sm.Current()).To(Equal("ERROR"))
	})
})
