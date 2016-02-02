/*
Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package crypto

import (
	"encoding/asn1"
	"errors"
	"github.com/openblockchain/obc-peer/openchain/crypto/ecies"
	"github.com/openblockchain/obc-peer/openchain/crypto/ecies/generic"
	"github.com/openblockchain/obc-peer/openchain/crypto/utils"
	obc "github.com/openblockchain/obc-peer/protos"
)

func (validator *validatorImpl) deepCloneAndDecryptTx1_2(tx *obc.Transaction) (*obc.Transaction, error) {
	if tx.Nonce == nil || len(tx.Nonce) == 0 {
		return nil, errors.New("Failed decrypting payload. Invalid nonce.")
	}

	// clone tx
	clone, err := validator.deepCloneTransaction(tx)
	if err != nil {
		validator.log.Error("Failed deep cloning [%s].", err.Error())
		return nil, err
	}

	var ccPrivateKey ecies.PrivateKey

	validator.log.Debug("Transaction type [%s].", tx.Type.String())

	validator.log.Debug("Extract transaction key...")

	// Derive transaction key
	es, err := generic.NewEncryptionSchemeFromPrivateKey(validator.chainPrivateKey)
	if err != nil {
		validator.log.Error("Failed init decryption engine [%s].", err.Error())
		return nil, err
	}

	validator.log.Debug("Decrypting message to validators [%s].", utils.EncodeBase64(tx.Key))

	msgToValidatorsRaw, err := es.Process(tx.Key)
	if err != nil {
		validator.log.Error("Failed decrypting transaction key [%s].", err.Error())
		return nil, err
	}

	msgToValidators := new(chainCodeValidatorMessage1_2)
	_, err = asn1.Unmarshal(msgToValidatorsRaw, msgToValidators)
	if err != nil {
		validator.log.Error("Failed unmarshalling message to validators [%s].", err.Error())
		return nil, err
	}

	validator.log.Debug("Deserializing transaction key [%s].", utils.EncodeBase64(msgToValidators.PrivateKey))
	ccPrivateKey, err = generic.DeserializePrivateKey(msgToValidators.PrivateKey)
	if err != nil {
		validator.log.Error("Failed deserializing transaction key [%s].", err.Error())
		return nil, err
	}

	validator.log.Debug("Extract transaction key...done")

	es, err = generic.NewEncryptionSchemeFromPrivateKey(ccPrivateKey)
	if err != nil {
		validator.log.Error("Failed init transaction decryption engine [%s].", err.Error())
		return nil, err
	}
	// Decrypt Payload
	payload, err := es.Process(clone.Payload)
	if err != nil {
		validator.log.Error("Failed decrypting payload [%s].", err.Error())
		return nil, err
	}
	clone.Payload = payload

	// Decrypt ChaincodeID
	chaincodeID, err := es.Process(clone.ChaincodeID)
	if err != nil {
		validator.log.Error("Failed decrypting chaincode [%s].", err.Error())
		return nil, err
	}
	clone.ChaincodeID = chaincodeID

	// Decrypt metadata
	if len(clone.Metadata) != 0 {
		metadata, err := es.Process(clone.Metadata)
		if err != nil {
			validator.log.Error("Failed decrypting metadata [%s].", err.Error())
			return nil, err
		}
		clone.Metadata = metadata
	}

	return clone, nil
}
