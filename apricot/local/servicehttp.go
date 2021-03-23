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
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"

    "github.com/AliceO2Group/Control/common/logger"
    "github.com/AliceO2Group/Control/configuration/cfgbackend"
    "github.com/AliceO2Group/Control/configuration/componentcfg"
    "github.com/AliceO2Group/Control/configuration/template"
    "github.com/flosch/pongo2/v4"
    "github.com/gorilla/mux"
    "github.com/sirupsen/logrus"
    "github.com/spf13/viper"
)

var log = logger.New(logrus.StandardLogger(), "confsys")

type Service struct {
    src cfgbackend.Source
}

type machineInfo struct {
    name string
}

type clusterInfo struct {
    FLPs []machineInfo
}

// Just a skeleton of a function for getting FLP names if relevant
/*func queryMachines(cluster *clusterInfo) error {
    for rows.Next() {
        machine := machineInfo{}
        err = rows.Scan(&machineInfo.name)
        if err != nil {
            return err
        }
        cluster.FLPs = append(cluster.FLPs, machine)
    }
    err = rows.Err()
    if err != nil {
        return err
    }
    return nil
}*/

func ApiGetClusterInformation(w http.ResponseWriter, r *http.Request) {
    queryParam := mux.Vars(r)
    format := ""
    var err error
    format, err = queryParam["format"]
    if err != nil {
        format = "text"
    }
    switch format {
    case "json":
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        //TODO: write JSON
        /* where to get info? depending on answer, could proceed like:

        cluster := clusterInfo{}
        err := queryMachines(&cluster)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            fmt.Fprintf(w, "Error, no result in cluster.")
        }
        answer, err := json.Marshal(cluster)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            fmt.Fprintf(w, "Error, result could not be given.")
        }
        fmt.Fprintf(w, string(answer))
        */
    case "text":
        w.Header().Set("Content-Type", "text/plain")
        w.WriteHeader(http.StatusOK)
        //TODO: write plain text
        /* something like: ???
        data, _ := ioutil.ReadAll(response.Body)
        fmt.Println(string(data))
        */
    default: 
        w.Header().Set("Content-Type", "text/plain")
        w.WriteHeader(http.StatusOK)
        //TODO: write plain text
        /* something like: ???
        data, _ := ioutil.ReadAll(response.Body)
        fmt.Println(string(data))
        */
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

func NewHttpService(service configuration.Service) (svc *Service, err error) {
    var src cfgbackend.Source
    src, err = cfgbackend.NewSource(uri)
    router := mux.NewRouter()
    webApi := router.PathPrefix("/inventory/flps").Subrouter()
    webApi.HandleFunc("/{format}", ApiGetClusterInformation).Methods(http.MethodGet)
    webApi.HandleFunc("", ApiUnhandledRequest).Methods(http.MethodPost)
    webApi.HandleFunc("", ApiUnhandledRequest).Methods(http.MethodPut)
    webApi.HandleFunc("", ApiUnhandledRequest).Methods(http.MethodDelete)
    webApi.HandleFunc("", ApiRequestNotFound)
    log.WithError(http.ListenAndServe(uri, router)).Fatal("Fatal error with Web API.")
    return &Service{
        src: src,
    }, err
}