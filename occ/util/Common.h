/*
 * === This file is part of ALICE O² ===
 *
 * Copyright 2020 CERN and copyright holders of ALICE O².
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

#ifndef OCC_COMMON_H
#define OCC_COMMON_H

#include <tuple>
#include <string>
#include <sstream>

#include <boost/algorithm/string.hpp>
#include <boost/property_tree/ini_parser.hpp>
#include <boost/property_tree/xml_parser.hpp>
#include <boost/property_tree/json_parser.hpp>

std::tuple<std::string, boost::property_tree::ptree> propMapEntryToPtree(const std::string& key, const std::string& value) {
    // If the returned key equals the input key, something went wrong

    std::vector<std::string> split;
    boost::split(split, key, std::bind(std::equal_to<>(), ':', std::placeholders::_1));
    if (split.size() != 3) {
        std::cout << "error processing ptree declaration for configuration payload: " << key;
        return std::make_tuple(key, boost::property_tree::ptree());
    }

    auto syntax = split[1];
    auto newKey = split[2];
    boost::property_tree::ptree ptree;
    auto stream = std::stringstream(value);

    try {
        if (syntax == "ini") {
            boost::property_tree::read_ini(stream, ptree);
        }
        else if (syntax == "json") {
            boost::property_tree::read_json(stream, ptree);
        }
        else if (syntax == "xml") {
            boost::property_tree::read_xml(stream, ptree);
        }
        else {
            std::cout << "error processing syntax declaration for configuration payload: " << key;
            return std::make_tuple(key, boost::property_tree::ptree());
        }
    }
    catch (std::exception &e) {
        std::cout << "error loading configuration payload into ptree for key: " << key << " error: " << e.what();
        return std::make_tuple(key, boost::property_tree::ptree());
    }

    return std::make_tuple(newKey, ptree);
}

#endif //OCC_COMMON_H
