/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2018 CERN and copyright holders of ALICE O².
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

package control

import (
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/coconut/protos"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/xlab/treeprint"
)

var(
	blue = color.New(color.FgHiBlue).SprintFunc()
	green = color.New(color.FgHiGreen).SprintFunc()
	yellow = color.New(color.FgHiYellow).SprintFunc()
	red = color.New(color.FgHiRed).SprintFunc()
	grey = color.New(color.FgWhite).SprintFunc()
)

func formatRunNumber(rn uint64) string {
	rnString := strconv.FormatUint(rn, 10)
	if rn == 0 {
		rnString = grey("none")
	} else {
		rnString = red(rnString)
	}
	return rnString
}

func colorState(st string) string {
	switch st {
	case "STANDBY", "DONE":
		return blue(st)
	case "RUNNING":
		return green(st)
	case "CONFIGURED":
		return yellow(st)
	default:
		return red(st)
	}
}

func colorGlobalState(st string) string {
	switch st {
	case "INITIAL", "FINAL":
		return yellow(st)
	case "CONNECTED":
		return green(st)
	default:
		return red(st)
	}
}


func colorStateFromNode(node *pb.RoleInfo) string {
	return colorState(node.GetState())
}

func buildTree(tree *treeprint.Tree, node *pb.RoleInfo, level int) {
	if len(node.Roles) != 0 {
		branch := (*tree).AddMetaBranch(colorStateFromNode(node), node.GetName())
		for _, n := range node.Roles {
			buildTree(&branch, n, level + 1)
		}
	} else {
		var nodeText string
		if len(node.GetTaskIds()) == 1 {
			// we format to include some padding and then the task
			yellow := color.New(color.FgHiYellow).SprintFunc()
			nodeText = fmt.Sprintf("%-"+strconv.Itoa(50-(4*level))+"s", node.GetName()) + yellow(" --> ") + "task " + node.GetTaskIds()[0]
		} else {
			nodeText = node.GetName()
		}
		(*tree).AddMetaNode(colorStateFromNode(node), nodeText)
	}
}

func drawWorkflow(root *pb.RoleInfo, o io.Writer) {
	if root == nil {
		return
	}
	tree := treeprint.New()

	tree.SetValue(root.GetName())
	if len(root.State) != 0 {
		tree.SetMetaValue(colorStateFromNode(root))
	}
	if len(root.Roles) != 0 {
		for _, n := range root.Roles {
			buildTree(&tree, n, 0)
		}
	}
	fmt.Fprint(o, tree.String())
}

type linePrintFunc func(t *pb.ShortTaskInfo) []string

func drawTableShortTaskInfos(tasks []*pb.ShortTaskInfo, headers []string, linePrint linePrintFunc, o io.Writer) {
	table := tablewriter.NewWriter(o)
	table.SetHeader(headers)
	table.SetBorder(false)
	fg := tablewriter.Colors{tablewriter.Bold, tablewriter.FgYellowColor}
	fgColSlice := make([]tablewriter.Colors, len(headers))
	for i := 0; i < len(headers); i++ {
		fgColSlice[i] = fg
	}
	table.SetHeaderColor(fgColSlice...)

	data := make([][]string, 0, 0)
	for _, taski := range tasks {
		data = append(data, linePrint(taski))
	}

	table.AppendBulk(data)
	table.Render()
}

func formatTimestamp(rfc3339timestamp string) string {
	timestamp, err := time.Parse(time.RFC3339, rfc3339timestamp)
	var formatted string
	if err == nil {
		formatted = timestamp.Local().Format("2006-01-02 15:04:05 MST")
	} else {
		formatted = "unknown"
	}
	return formatted
}
