// Copyright (C) 2019-2025 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"log"

	"github.com/algorand/go-algorand/data/basics"
	"github.com/algorand/go-algorand/protocol"

	v2 "github.com/algorand/go-algorand/daemon/algod/api/server/v2"
	"github.com/algorand/go-algorand/daemon/algod/api/server/v2/generated/model"
)

// ddrFromParams converts serialized DryrunRequest to v2.DryrunRequest
func ddrFromParams(dp *DebugParams) (ddr v2.DryrunRequest, err error) {
	if len(dp.DdrBlob) == 0 {
		return
	}

	var gdr model.DryrunRequest
	err1 := protocol.DecodeJSON(dp.DdrBlob, &gdr)
	if err1 == nil {
		ddr, err = v2.DryrunRequestFromGenerated(&gdr)
	} else {
		err = protocol.DecodeReflect(dp.DdrBlob, &ddr)
		// if failed report intermediate decoding error
		if err != nil {
			log.Printf("Decoding as JSON DryrunRequest object failed: %s", err1.Error())
		}
	}

	return
}

func balanceRecordsFromDdr(ddr *v2.DryrunRequest) (records []basics.BalanceRecord, err error) {
	accounts := make(map[basics.Address]basics.AccountData)
	for _, a := range ddr.Accounts {
		var addr basics.Address
		addr, err = basics.UnmarshalChecksumAddress(a.Address)
		if err != nil {
			return
		}
		var ad basics.AccountData
		ad, err = v2.AccountToAccountData(&a)
		if err != nil {
			return
		}
		accounts[addr] = ad
	}
	for _, a := range ddr.Apps {
		var addr basics.Address
		addr, err = basics.UnmarshalChecksumAddress(a.Params.Creator)
		if err != nil {
			return
		}
		// deserialize app params and update account data
		var params basics.AppParams
		params, err = v2.ApplicationParamsToAppParams(&a.Params)
		if err != nil {
			return
		}
		ad := accounts[addr]
		if ad.AppParams == nil {
			ad.AppParams = make(map[basics.AppIndex]basics.AppParams, 1)
			ad.AppParams[a.Id] = params
		} else {
			ap, ok := ad.AppParams[a.Id]
			if ok {
				v2.MergeAppParams(&ap, &params)
				ad.AppParams[a.Id] = ap
			} else {
				ad.AppParams[a.Id] = params
			}
		}
		accounts[addr] = ad
	}

	for addr, ad := range accounts {
		records = append(records, basics.BalanceRecord{Addr: addr, AccountData: ad})
	}
	return
}
