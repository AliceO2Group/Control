/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019-2021 CERN and copyright holders of ALICE O².
 * Author: Claire Guyot <claire.eloise.guyot@cern.ch>
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

package local

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AliceO2Group/Control/common/system"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

type HttpService struct {
	svc configuration.Service
}

func (httpsvc *HttpService) ApiGetFlps(w http.ResponseWriter, r *http.Request) {
	httpsvc.ApiGetClusterInformation(w, r, "")
}

func (httpsvc *HttpService) ApiGetDetectorFlps(w http.ResponseWriter, r *http.Request) {
	queryParam := mux.Vars(r)
	detector := queryParam["detector"]
	_, err := system.IDString(detector)
	if err != nil {
		log.WithError(err).Warn("Error, the detector name provided is not valid.")
	} else {
		httpsvc.ApiGetClusterInformation(w, r, detector)
	}
}

func (httpsvc *HttpService) ApiGetClusterInformation(w http.ResponseWriter, r *http.Request, detector string) {
	queryParam := mux.Vars(r)
	format := ""
	format = queryParam["format"]
	if format == "" {
		format = "text"
	}
	keys, err := httpsvc.svc.GetHostInventory(detector)
	if err != nil {
		log.WithError(err).Warn("Error, could not retrieve host list.")
	}
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		hosts, err := json.MarshalIndent(keys, "", "\t")
		if err != nil {
			log.WithError(err).Warn("Error, could not marshal hosts.")
		}
		fmt.Fprintln(w, string(hosts))
	case "text": fallthrough
	default:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		for _, hostname := range keys {
			fmt.Fprintf(w,"%s\n", hostname)
		}
	}
}

func NewHttpService(service configuration.Service) (svr *http.Server) {
	router := mux.NewRouter()
	httpsvc := &HttpService{
		svc: service,
	}
	httpsvr := &http.Server{
		Handler:      router,
		Addr:         ":" + strconv.Itoa(viper.GetInt("httpListenPort")),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	apiFlps := router.PathPrefix("/inventory/flps").Subrouter()
	apiFlps.HandleFunc("", httpsvc.ApiGetFlps).Methods(http.MethodGet)
	apiFlps.HandleFunc("/", httpsvc.ApiGetFlps).Methods(http.MethodGet)
	apiFlps.HandleFunc("/{format}", httpsvc.ApiGetFlps).Methods(http.MethodGet)
	apiDetectorFlps := router.PathPrefix("/inventory/detectors/{detector}/flps").Subrouter()
	apiDetectorFlps.HandleFunc("", httpsvc.ApiGetDetectorFlps).Methods(http.MethodGet)
	apiDetectorFlps.HandleFunc("/", httpsvc.ApiGetDetectorFlps).Methods(http.MethodGet)
	apiDetectorFlps.HandleFunc("/{format}", httpsvc.ApiGetDetectorFlps).Methods(http.MethodGet)

	// async-start of http Service and capture error
	go func() {
		err := httpsvr.ListenAndServe()
		if err != nil {
			log.WithError(err).Error("HTTP service returned error")
		}
	}()
	return httpsvr
}
