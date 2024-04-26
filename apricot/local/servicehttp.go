/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2019-2023 CERN and copyright holders of ALICE O².
 * Author: Claire Guyot <claire.eloise.guyot@cern.ch>
 * Author: Teo Mrnjavac <teo.m@cern.ch>
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
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/AliceO2Group/Control/apricot/docs"
	apricotpb "github.com/AliceO2Group/Control/apricot/protos"
	"github.com/AliceO2Group/Control/common/logger/infologger"
	"github.com/AliceO2Group/Control/common/system"
	"github.com/AliceO2Group/Control/configuration"
	"github.com/AliceO2Group/Control/configuration/componentcfg"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type HttpService struct {
	svc configuration.Service
}

//	@title			O² Apricot REST API
//	@version		1.0
//	@description	REST API for ALICE O² Apricot configuration service

//	@contact.name	O² FLP support
//	@contact.url	https://alice-flp.docs.cern.ch/
//	@contact.email	alice-o2-flp-support@cern.ch

//	@externalDocs.description	AliECS handbook
//	@externalDocs.url			https://alice-flp.docs.cern.ch/aliecs/handbook/

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

	// documentation endpoint
	_ = router.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)

	// component configuration API

	// GET /components
	apiComponents := router.PathPrefix("/components").Subrouter()
	apiComponents.HandleFunc("", httpsvc.ApiListComponents).Methods(http.MethodGet)
	apiComponents.HandleFunc("/", httpsvc.ApiListComponents).Methods(http.MethodGet)
	// POST /components/_invalidate_cache
	apiComponents.HandleFunc("/_invalidate_cache", httpsvc.ApiInvalidateCache).Methods(http.MethodPost)

	// GET /components/{component}
	apiComponentsEntries := router.PathPrefix("/components/{component}").Subrouter()
	// GET /components/{component} returns all, raw is ignored
	apiComponentsEntries.HandleFunc("", httpsvc.ApiListComponentEntries).Methods(http.MethodGet)
	apiComponentsEntries.HandleFunc("/", httpsvc.ApiListComponentEntries).Methods(http.MethodGet)
	// runtype = {runtype} rolename = any, raw excludes ANY runtype, if false returns all
	apiComponentsEntries.HandleFunc("/{runtype}", httpsvc.ApiListComponentEntries).Methods(http.MethodGet)
	apiComponentsEntries.HandleFunc("/{runtype}/", httpsvc.ApiListComponentEntries).Methods(http.MethodGet)
	// runtype = {runtype} rolename = {rolename}, raw excludes ANY runtype and any rolename, if false returns all
	apiComponentsEntries.HandleFunc("/{runtype}/{rolename}", httpsvc.ApiListComponentEntries).Methods(http.MethodGet)
	apiComponentsEntries.HandleFunc("/{runtype}/{rolename}/", httpsvc.ApiListComponentEntries).Methods(http.MethodGet)

	apiComponentQuery := router.PathPrefix("/components/{component}/{runtype}/{rolename}/{entry}").Subrouter()
	// GET /components/{component}/{runtype}/{rolename}/{entry}/resolve, assumes this is not a raw path, returns a raw path
	// like {component}/{runtype}/{rolename}/{entry}
	apiComponentQuery.HandleFunc("/resolve", httpsvc.ApiResolveComponentQuery).Methods(http.MethodGet)
	apiComponentQuery.HandleFunc("/{subentry}/resolve", httpsvc.ApiResolveComponentQuery).Methods(http.MethodGet)
	// GET /components/{component}/{runtype}/{rolename}/{entry}, accepts raw or non-raw path, returns payload
	// that may be processed or not depending on process=true or false
	apiComponentQuery.HandleFunc("", httpsvc.ApiGetComponentConfiguration).Methods(http.MethodGet)
	apiComponentQuery.HandleFunc("/", httpsvc.ApiGetComponentConfiguration).Methods(http.MethodGet)
	apiComponentQuery.HandleFunc("/{subentry}", httpsvc.ApiGetComponentConfiguration).Methods(http.MethodGet)

	// inventory API

	apiInventoryFlps := router.PathPrefix("/inventory/flps").Subrouter()
	apiInventoryFlps.HandleFunc("", httpsvc.ApiGetFlps).Methods(http.MethodGet)
	apiInventoryFlps.HandleFunc("/", httpsvc.ApiGetFlps).Methods(http.MethodGet)
	apiInventoryFlps.HandleFunc("/{format}", httpsvc.ApiGetFlps).Methods(http.MethodGet)

	apiInventoryDetectors := router.PathPrefix("/inventory/detectors").Subrouter()
	apiInventoryDetectors.HandleFunc("", httpsvc.ApiGetDetectorsInventory).Methods(http.MethodGet)
	apiInventoryDetectors.HandleFunc("/", httpsvc.ApiGetDetectorsInventory).Methods(http.MethodGet)
	apiInventoryDetectors.HandleFunc("/{format}", httpsvc.ApiGetDetectorsInventory).Methods(http.MethodGet)

	apiInventoryDetectorFlps := router.PathPrefix("/inventory/detectors/{detector}/flps").Subrouter()
	apiInventoryDetectorFlps.HandleFunc("", httpsvc.ApiGetDetectorFlps).Methods(http.MethodGet)
	apiInventoryDetectorFlps.HandleFunc("/", httpsvc.ApiGetDetectorFlps).Methods(http.MethodGet)
	apiInventoryDetectorFlps.HandleFunc("/{format}", httpsvc.ApiGetDetectorFlps).Methods(http.MethodGet)

	// async-start of http Service and capture error
	go func() {
		log.WithField("port", viper.GetInt("httpListenPort")).
			WithField("level", infologger.IL_Support).
			Info("HTTP service started")

		err := httpsvr.ListenAndServe()
		if err != nil {
			log.WithError(err).Error("HTTP service returned error")
		}
	}()
	return httpsvr
}

// ApiListComponents lists configuration components
//
//	@Summary		List Apricot-managed configuration components
//	@Description	Returns a list of all configuration components managed by Apricot
//	@Tags			component configuration
//	@Produce		json
//	@Produce		plain
//	@Param			format	query		string	false	"Output format, json or text"	Enums(json, text)	Default(text)
//	@Success		200		{array}		string	"List of components, either as JSON array or comma-separated plain text"
//	@Failure		500		{string}	string	"Internal server error"
//	@Router			/components [get]
func (httpsvc *HttpService) ApiListComponents(w http.ResponseWriter, r *http.Request) {
	queryArgs := r.URL.Query()
	format := queryArgs.Get("format")
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		components, err := httpsvc.svc.ListComponents()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			out, _ := json.MarshalIndent(err, "", "\t")
			_, _ = fmt.Fprintln(w, string(out))
			return
		}

		sort.Strings(components)

		response, err := json.MarshalIndent(components, "", "\t")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			out, _ := json.MarshalIndent(err, "", "\t")
			_, _ = fmt.Fprintln(w, string(out))
			return
		}
		_, _ = fmt.Fprintln(w, string(response))
		return

	case "text":
		fallthrough
	default:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		components, err := httpsvc.svc.ListComponents()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintln(w, err)
			return
		}

		sort.Strings(components)

		response := strings.Join(components, "\n")
		_, _ = fmt.Fprintln(w, string(response))
	}
}

// ApiInvalidateCache clears the component template cache
//
//	@Summary		Clear cache of Apricot-managed configuration component templates
//	@Description	Invalidates all cached templates for all configuration components managed by Apricot
//	@Tags			component configuration
//	@Produce		plain
//	@Success		200
//	@Router			/components/_invalidate_cache [post]
func (httpsvc *HttpService) ApiInvalidateCache(w http.ResponseWriter, r *http.Request) {
	log.WithField("level", infologger.IL_Support).
		Debug("invalidating component template cache")
	httpsvc.svc.InvalidateComponentTemplateCache()
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "OK")
}

// ApiListComponentEntries lists all entries for a given configuration component
//
//	@Summary		Lists all entries for an Apricot-managed configuration component
//	@Description	Returns a list of all entries belonging to a given configuration component corresponding to the given query. The component must be provided. If the run type is not provided, entries for all run types are returned. If the role name is not provided, entries for all role names are returned. If the raw flag is true, AApricot will not attempt to include "ANY"/"any" run types or role names, and will return only the entries stored under the provided Consul path. If the raw flag is false, Apricot will include all entries for the given component, run type and role name, including entries stored under "ANY" and "any" paths corresponding to fallback values valid for all run types or role names.
//	@Tags			component configuration
//	@Produce		json
//	@Produce		plain
//	@Param			format		query		string	false	"Output format, json or text"																					Enums(json, text)	Default(text)
//	@Param			raw			query		boolean	false	"If true, the query returns strictly Consul paths without catching all run types or role names with ANY/any"	Default(false)
//	@Param			component	path		string	true	"Configuration component"
//	@Param			runtype		path		string	false	"O² Run type, must be capitalized"	Default(ANY)
//	@Param			rolename	path		string	false	"Role name"							Default(any)
//	@Success		200			{array}		string	"List of entries, either as JSON array or comma-separated plain text"
//	@Failure		400			{string}	string	"Bad request, if the run type is invalid"
//	@Failure		500			{string}	string	"Internal server error"
//	@Router			/components/{component} [get]
//	@Router			/components/{component}/{runtype} [get]
//	@Router			/components/{component}/{runtype}/{rolename} [get]
func (httpsvc *HttpService) ApiListComponentEntries(w http.ResponseWriter, r *http.Request) {
	// GET /components/{component} returns all, raw is ignored
	// runtype = {runtype} rolename = any, raw excludes ANY runtype, if false returns all
	// runtype = {runtype} rolename = {rolename}, raw excludes ANY runtype and any rolename, if false returns all

	queryParams := mux.Vars(r)
	component, hasComponent := queryParams["component"]
	if !hasComponent {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "component name not provided")
		return
	}

	runtypeS, hasRuntype := queryParams["runtype"]

	runType := apricotpb.RunType_NULL
	if hasRuntype {
		runtypeS = strings.ToUpper(runtypeS)
		runTypeInt, isRunTypeValid := apricotpb.RunType_value[runtypeS]
		if !isRunTypeValid {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintln(w, "runtype not valid")
			return
		} else {
			runType = apricotpb.RunType(runTypeInt)
		}
	}

	rolename, hasRolename := queryParams["rolename"]

	queryArgs := r.URL.Query()
	rawS := queryArgs.Get("raw")
	raw, err := strconv.ParseBool(rawS)
	if err != nil {
		raw = false
	}

	entries, err := httpsvc.svc.ListComponentEntries(&componentcfg.EntriesQuery{
		Component: component,
		RunType:   runType,
		RoleName:  rolename,
	}, false)

	filterPrefixes := make(map[string]struct{})
	if hasRuntype {
		if hasRolename { // we filter for runtype and rolename, and if !raw, also combos with ANY and any
			filterPrefixes[runtypeS+"/"+rolename] = struct{}{}
			if !raw {
				filterPrefixes["ANY"+"/"+rolename] = struct{}{}
				filterPrefixes[runtypeS+"/any"] = struct{}{}
				filterPrefixes["ANY/any"] = struct{}{}
			}
		} else { // no rolename provided, we only filter for runtype and ANY if !raw
			filterPrefixes[runtypeS] = struct{}{}
			if !raw {
				filterPrefixes["ANY"] = struct{}{}
			}
		}
	}

	filteredEntries := make([]string, 0)
	if hasRuntype { // if there's any filtering to do
		for _, entry := range entries {
			for filterPrefix, _ := range filterPrefixes {
				if strings.HasPrefix(entry, filterPrefix) {
					filteredEntries = append(filteredEntries, entry)
				}
			}
		}
	} else { // no filtering, return all entries
		filteredEntries = entries
	}

	format := queryArgs.Get("format")

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			out, _ := json.MarshalIndent(err, "", "\t")
			_, _ = fmt.Fprintln(w, string(out))
			return
		}

		w.WriteHeader(http.StatusOK)

		sort.Strings(filteredEntries)

		response, err := json.MarshalIndent(filteredEntries, "", "\t")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			out, _ := json.MarshalIndent(err, "", "\t")
			_, _ = fmt.Fprintln(w, string(out))
			return
		}
		_, _ = fmt.Fprintln(w, string(response))
		return

	case "text":
		fallthrough
	default:
		w.Header().Set("Content-Type", "text/plain")

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintln(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)

		sort.Strings(filteredEntries)

		response := strings.Join(filteredEntries, "\n")
		_, _ = fmt.Fprintln(w, string(response))
	}
}

// ApiResolveComponentQuery resolves a query for a given component, run type, role name and entry key
//
//	@Summary		Resolves a query for a given component, run type, role name and entry key
//	@Description	Returns a resolved path for a given component, run type, role name and entry key. The path points to an actual existing entry in Consul, resolving ANY run type and any rolename wildcards.
//	@Tags			component configuration
//	@Produce		plain
//	@Param			component	path		string	true	"Configuration component"
//	@Param			runtype		path		string	true	"O² Run type, must be capitalized"
//	@Param			rolename	path		string	true	"Role name"
//	@Param			entry		path		string	true	"Entry key"
//	@Success		200			{string}	string	"Resolved path for the queried entry"
//	@Failure		400			{string}	string	"Bad request, if a parameter is invalid"
//	@Failure		500			{string}	string	"Internal server error"
//	@Router			/components/{component}/{runtype}/{rolename}/{entry}/resolve [get]
func (httpsvc *HttpService) ApiResolveComponentQuery(w http.ResponseWriter, r *http.Request) {
	// GET /components/{component}/{runtype}/{rolename}/{entry}/resolve, assumes this is not a raw path, returns a raw path
	// like {component}/{runtype}/{rolename}/{entry}

	queryParams := mux.Vars(r)
	component, hasComponent := queryParams["component"]
	if !hasComponent {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "component name not provided")
		return
	}

	runtypeS, hasRuntype := queryParams["runtype"]
	runType := apricotpb.RunType_NULL
	if hasRuntype {
		runtypeS = strings.ToUpper(runtypeS)
		runTypeInt, isRunTypeValid := apricotpb.RunType_value[runtypeS]
		if !isRunTypeValid {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintln(w, "runtype not valid")
			return
		} else {
			runType = apricotpb.RunType(runTypeInt)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "runtype not provided")
		return
	}

	rolename, hasRolename := queryParams["rolename"]
	if !hasRolename {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "rolename not provided")
		return
	}

	entry, hasEntry := queryParams["entry"]
	if !hasEntry {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "entry not provided")
		return
	}

	subentry, hasSubentry := queryParams["subentry"]
	if hasSubentry {
		entry = entry + "/" + subentry
	}

	resolved, err := httpsvc.svc.ResolveComponentQuery(&componentcfg.Query{
		Component: component,
		RunType:   runType,
		RoleName:  rolename,
		EntryKey:  entry,
		Timestamp: "",
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, err)
		return
	}

	resolvedStr := resolved.Path()

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, resolvedStr)
}

// ApiGetComponentConfiguration returns the processed configuration payload for a given component, run type, role name and entry key
//
//	@Summary		Returns a configuration payload for a given component, run type, role name and entry key
//	@Description	The provided component, run type, role name and entry key are used to query the configuration service for a configuration entry, which is then processed in the O² Apricot template system to produce the final payload, and returned as string..
//	@Tags			component configuration
//	@Produce		plain
//	@Param			process		query		boolean	false	"If true, template processing is performed to produce the final payload; if false, the entry is returned verbatim. In the true case, any number of additional string key-value pairs may be passed as query parameters (e.g. ?process=true&mykey1=myvalue1&mykey2=myvalue2), which are then fed into the template system as variables that affect configuration payload generation"	Default(false)
//	@Param			component	path		string	true	"Configuration component"
//	@Param			runtype		path		string	true	"O² Run type, must be capitalized"
//	@Param			rolename	path		string	true	"Role name"
//	@Param			entry		path		string	true	"Entry key"
//	@Success		200			{string}	string	"Configuration payload for the queried entry, either processed or verbatim depending on the process flag"
//	@Failure		400			{string}	string	"Bad request, if a parameter is invalid"
//	@Failure		500			{string}	string	"Internal server error"
//	@Router			/components/{component}/{runtype}/{rolename}/{entry} [get]
func (httpsvc *HttpService) ApiGetComponentConfiguration(w http.ResponseWriter, r *http.Request) {
	// GET /components/{component}/{runtype}/{rolename}/{entry}, accepts raw or non-raw path, returns payload
	// that may be processed or not depending on process=true or false

	queryParams := mux.Vars(r)
	component, hasComponent := queryParams["component"]
	if !hasComponent {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "component name not provided")
		return
	}

	runtypeS, hasRuntype := queryParams["runtype"]
	runType := apricotpb.RunType_NULL
	if hasRuntype {
		runtypeS = strings.ToUpper(runtypeS)
		runTypeInt, isRunTypeValid := apricotpb.RunType_value[runtypeS]
		if !isRunTypeValid {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintln(w, "runtype not valid")
			return
		} else {
			runType = apricotpb.RunType(runTypeInt)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "runtype not provided")
		return
	}

	rolename, hasRolename := queryParams["rolename"]
	if !hasRolename {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "rolename not provided")
		return
	}

	entry, hasEntry := queryParams["entry"]
	if !hasEntry {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintln(w, "entry not provided")
		return
	}

	subentry, hasSubentry := queryParams["subentry"]
	if hasSubentry {
		entry = entry + "/" + subentry
	}

	queryArgs := r.URL.Query()
	processS := queryArgs.Get("process")
	process, err := strconv.ParseBool(processS)
	if err != nil {
		process = false
	}
	if queryArgs.Has("process") {
		queryArgs.Del("process")
	}

	varStack := make(map[string]string)
	for k, v := range queryArgs {
		if len(v) > 0 {
			varStack[k] = v[0]
		}
	}

	var payload string
	if process {
		payload, err = httpsvc.svc.GetAndProcessComponentConfiguration(&componentcfg.Query{
			Component: component,
			RunType:   runType,
			RoleName:  rolename,
			EntryKey:  entry,
		}, varStack)
	} else {
		payload, err = httpsvc.svc.GetComponentConfiguration(&componentcfg.Query{
			Component: component,
			RunType:   runType,
			RoleName:  rolename,
			EntryKey:  entry,
		})
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintln(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, payload)
}

// ApiGetFlps returns the list of FLPs in the cluster known to Apricot
//
//	@Summary		Returns the list of FLPs in the cluster known to Apricot
//	@Description	Returns the list of all Apricot-managed hosts in the cluster that are known to be FLPs, newline-separated or JSON depending on the format parameter
//	@Tags			cluster inventory
//	@Param			format	path	string	false	"Output format, json or text"	Enums(json, text)	Default(text)
//	@Produce		plain
//	@Success		200	{string}	string	"List of FLPs in the cluster known to Apricot, newline-separated or JSON depending on the format parameter"
//	@Router			/inventory/flps/{format} [get]
func (httpsvc *HttpService) ApiGetFlps(w http.ResponseWriter, r *http.Request) {
	httpsvc.ApiGetHostInventory(w, r, "")
}

// ApiGetDetectorFlps returns the list of FLPs in the cluster that serve a given detector
//
//	@Summary		Returns the list of FLPs in the cluster that serve a given detector
//	@Description	Returns the list of all Apricot-managed hosts in the cluster that are known to be FLPs and serving the given detector, newline-separated or JSON depending on the format parameter
//	@Tags			cluster inventory
//	@Param			format	path	string	false	"Output format, json or text"	Enums(json, text)	Default(text)
//	@Produce		plain
//	@Success		200	{string}	string	"List of FLPs in the cluster known to Apricot, newline-separated or JSON depending on the format parameter"
//	@Router			/inventory/detectors/{detector}/flps/{format} [get]
func (httpsvc *HttpService) ApiGetDetectorFlps(w http.ResponseWriter, r *http.Request) {
	queryParam := mux.Vars(r)
	detector := queryParam["detector"]
	_, err := system.IDString(detector)
	if err != nil {
		log.WithError(err).Warn("Error, the detector name provided is not valid.")
	} else {
		httpsvc.ApiGetHostInventory(w, r, detector)
	}
}

func (httpsvc *HttpService) ApiGetHostInventory(w http.ResponseWriter, r *http.Request, detector string) {
	hosts, err := httpsvc.svc.GetHostInventory(detector)
	if err != nil {
		log.WithError(err).Warn("Error, could not retrieve host list.")
	}
	httpsvc.ApiPrintClusterInformation(w, r, hosts, nil)
}

// ApiGetDetectorsInventory returns the list of detectors belonging to the installed instance known to Apricot
//
//	@Summary		Returns the list of detectors belonging to the installed instance known to Apricot
//	@Description	Returns the list of all detectors known to Apricot that belong to the installed instance, newline-separated or JSON depending on the format parameter
//	@Tags			cluster inventory
//	@Param			format	path	string	false	"Output format, json or text"	Enums(json, text)	Default(text)
//	@Produce		plain
//	@Success		200	{string}	string	"List of detectors in the instance known to Apricot, newline-separated or JSON depending on the format parameter"
//	@Router			/inventory/detectors/{format} [get]
func (httpsvc *HttpService) ApiGetDetectorsInventory(w http.ResponseWriter, r *http.Request) {
	inventory, err := httpsvc.svc.GetDetectorsInventory()
	if err != nil {
		log.WithError(err).Warn("Error, could not retrieve detectors inventory list.")
	}
	httpsvc.ApiPrintClusterInformation(w, r, nil, inventory)
}

func (httpsvc *HttpService) ApiPrintClusterInformation(w http.ResponseWriter, r *http.Request, hosts []string, inventory map[string][]string) {
	queryParam := mux.Vars(r)
	format := ""
	format = queryParam["format"]
	if format == "" {
		format = "text"
	}
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		var result []byte
		var err error
		if hosts != nil {
			result, err = json.MarshalIndent(hosts, "", "\t")
			if err != nil {
				log.WithError(err).Warn("Error, could not marshal hosts.")
			}
		} else if inventory != nil {
			result, err = json.MarshalIndent(inventory, "", "\t")
			if err != nil {
				log.WithError(err).Warn("Error, could not marshal inventory.")
			}
		}
		fmt.Fprintln(w, string(result))
	case "text":
		fallthrough
	default:
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		if hosts != nil {
			for _, hostname := range hosts {
				fmt.Fprintf(w, "%s\n", hostname)
			}
		} else if inventory != nil {
			for detector, flps := range inventory {
				fmt.Fprintf(w, "%s\n", detector)
				for _, hostname := range flps {
					fmt.Fprintf(w, "\t%s\n", hostname)
				}
			}
		}
	}
}
