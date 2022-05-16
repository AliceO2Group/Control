/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2021 CERN and copyright holders of ALICE O².
 * Author: Kostas Alexopoulos <kostas.alexopoulos@cern.ch>
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

package repos

import (
	"net/url"
	"path"
)

type httpsRepo struct {
	Repo
}

func (r *httpsRepo) getUri() string {
	u, _ := url.Parse(r.GetProtocol() + "://")
	u.Path = path.Join(u.Path, r.HostingSite)

	u.Path = path.Join(u.Path,
		r.Path,
		r.RepoName)

	uri := u.String()

	return uri + ".git"
}

func (r *httpsRepo) GetProtocol() string {
	return "https"
}

/*func (r *httpsRepo) GetDplCommand(dplCommandUri string) (string, error) {
	dplCommandPath := filepath.Join(r.GetCloneDir(), jitScriptsDir, dplCommandUri)
	dplCommandPayload, err := os.ReadFile(dplCommandPath)
	if err != nil {
		return "", err
	}
	return string(dplCommandPayload), nil
}*/
