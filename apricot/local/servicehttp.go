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
	"time"

	"github.com/AliceO2Group/Control/configuration"
	"github.com/gorilla/mux"
)

type HttpService struct {
	svc configuration.Service
}

type machineInfo struct {
	name string
}

type clusterInfo struct {
	FLPs []machineInfo
}

func (httpsvc *HttpService) ApiGetClusterInformation(w http.ResponseWriter, r *http.Request) {
	queryParam := mux.Vars(r)
	format := ""
	format = queryParam["format"]
	if format == "" {
		format = "text"
	}
	keys, err := httpsvc.svc.GetHostInventory()
	if err != nil {
		log.WithError(err).Fatal("Error, could not retrieve host list.")
	}
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		hosts, err := json.MarshalIndent(keys, "", "\t")
		if err != nil {
			log.WithError(err).Fatal("Error, could not marshal hosts.")
		}
		fmt.Fprintln(w, string(hosts))
	case "text":
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		for _, hostname := range keys {
			fmt.Fprintf(w,"%s\n", hostname)
		}
	default:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		for _, hostname := range keys {
			fmt.Fprintf(w,"%s\n", hostname)
		}
	}
}

func ApiUnhandledRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Request not handled. Request should be of type GET.")
}

func ApiRequestNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Request not found.")
}

func NewHttpService(service configuration.Service) (svr *http.Server) {
	router := mux.NewRouter()
	httpsvc := &HttpService{
		svc: service,
	}
	httpsvr := &http.Server{
		Handler:      router,
		Addr:         ":47188",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	go func() {
		webApi := router.PathPrefix("/inventory/flps").Subrouter()
		webApi.HandleFunc("/", httpsvc.ApiGetClusterInformation).Methods(http.MethodGet)
		webApi.HandleFunc("/{format}", httpsvc.ApiGetClusterInformation).Methods(http.MethodGet)
		webApi.HandleFunc("/", ApiUnhandledRequest).Methods(http.MethodPost)
		webApi.HandleFunc("/", ApiUnhandledRequest).Methods(http.MethodPut)
		webApi.HandleFunc("/", ApiUnhandledRequest).Methods(http.MethodDelete)
		webApi.HandleFunc("", ApiRequestNotFound)
		log.WithError(httpsvr.ListenAndServe()).Fatal("Fatal error with Http Service.")
	}()
	return httpsvr
}
