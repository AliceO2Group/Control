/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2025 CERN and copyright holders of ALICE O².
 * Author: Teo Mrnjavac <teo.mrnjavac@cern.ch>
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

package task

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("task status", func() {
	When("product of all statuses with UNDEFINED is done", func() {
		undefined := Status(UNDEFINED)
		It("should be UNDEFINED for all", func() {
			Expect(Status(UNDEFINED)).To(Equal(undefined.X(UNDEFINED)))
			Expect(Status(UNDEFINED)).To(Equal(undefined.X(INACTIVE)))
			Expect(Status(UNDEFINED)).To(Equal(undefined.X(PARTIAL)))
			Expect(Status(UNDEFINED)).To(Equal(undefined.X(ACTIVE)))
			Expect(Status(UNDEFINED)).To(Equal(undefined.X(UNDEPLOYABLE)))
			Expect(Status(UNDEFINED)).To(Equal(undefined.X(INVARIANT)))
		})
	})
	When("product of all statuses with INACTIVE is done", func() {
		inactive := Status(INACTIVE)
		It("should have different results", func() {
			Expect(Status(UNDEFINED)).To(Equal(inactive.X(UNDEFINED)))
			Expect(Status(INACTIVE)).To(Equal(inactive.X(INACTIVE)))
			Expect(Status(PARTIAL)).To(Equal(inactive.X(PARTIAL)))
			Expect(Status(PARTIAL)).To(Equal(inactive.X(ACTIVE)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(inactive.X(UNDEPLOYABLE)))
			Expect(Status(INACTIVE)).To(Equal(inactive.X(INVARIANT)))
		})
	})
	When("product of all statuses with PARTIAL is done", func() {
		partial := Status(PARTIAL)
		It("should have different results", func() {
			Expect(Status(UNDEFINED)).To(Equal(partial.X(UNDEFINED)))
			Expect(Status(PARTIAL)).To(Equal(partial.X(INACTIVE)))
			Expect(Status(PARTIAL)).To(Equal(partial.X(PARTIAL)))
			Expect(Status(PARTIAL)).To(Equal(partial.X(ACTIVE)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(partial.X(UNDEPLOYABLE)))
			Expect(Status(PARTIAL)).To(Equal(partial.X(INVARIANT)))
		})
	})
	When("product of all statuses with ACTIVE is done", func() {
		active := Status(ACTIVE)
		It("should have different results", func() {
			Expect(Status(UNDEFINED)).To(Equal(active.X(UNDEFINED)))
			Expect(Status(PARTIAL)).To(Equal(active.X(INACTIVE)))
			Expect(Status(PARTIAL)).To(Equal(active.X(PARTIAL)))
			Expect(Status(ACTIVE)).To(Equal(active.X(ACTIVE)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(active.X(UNDEPLOYABLE)))
			Expect(Status(ACTIVE)).To(Equal(active.X(INVARIANT)))
		})
	})
	When("product of all statuses with UNDEPLOYABLE is done", func() {
		undeployable := Status(UNDEPLOYABLE)
		It("should be UNDEPLOYABLE unless UNDEFINED", func() {
			Expect(Status(UNDEFINED)).To(Equal(undeployable.X(UNDEFINED)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(undeployable.X(INACTIVE)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(undeployable.X(PARTIAL)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(undeployable.X(ACTIVE)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(undeployable.X(UNDEPLOYABLE)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(undeployable.X(INVARIANT)))
		})
	})
	When("product of all statuses with INVARIANT is done", func() {
		invariant := Status(INVARIANT)
		It("it should be the status the product was done with", func() {
			Expect(Status(UNDEFINED)).To(Equal(invariant.X(UNDEFINED)))
			Expect(Status(INACTIVE)).To(Equal(invariant.X(INACTIVE)))
			Expect(Status(PARTIAL)).To(Equal(invariant.X(PARTIAL)))
			Expect(Status(ACTIVE)).To(Equal(invariant.X(ACTIVE)))
			Expect(Status(UNDEPLOYABLE)).To(Equal(invariant.X(UNDEPLOYABLE)))
			Expect(Status(INVARIANT)).To(Equal(invariant.X(INVARIANT)))
		})
	})
	When("String() of Status is called", func() {
		It("should return string representation of a status", func() {
			Expect("UNDEFINED").To(Equal(Status(UNDEFINED).String()))
			Expect("INACTIVE").To(Equal(Status(INACTIVE).String()))
			Expect("PARTIAL").To(Equal(Status(PARTIAL).String()))
			Expect("ACTIVE").To(Equal(Status(ACTIVE).String()))
			Expect("UNDEPLOYABLE").To(Equal(Status(UNDEPLOYABLE).String()))
			Expect("INVARIANT").To(Equal(Status(INVARIANT).String()))
			Expect("UNDEFINED").To(Equal(Status(255).String()))
		})
	})
})
