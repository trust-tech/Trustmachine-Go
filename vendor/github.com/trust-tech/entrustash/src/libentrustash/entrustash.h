/*
  This file is part of entrustash.

  entrustash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  entrustash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with entrustash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file entrustash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define ENTRUSTASH_REVISION 23
#define ENTRUSTASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define ENTRUSTASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define ENTRUSTASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define ENTRUSTASH_CACHE_BYTES_GROWTH 131072U  // 2**17
#define ENTRUSTASH_EPOCH_LENGTH 30000U
#define ENTRUSTASH_MIX_BYTES 128
#define ENTRUSTASH_HASH_BYTES 64
#define ENTRUSTASH_DATASET_PARENTS 256
#define ENTRUSTASH_CACHE_ROUNDS 3
#define ENTRUSTASH_ACCESSES 64
#define ENTRUSTASH_DAG_MAGIC_NUM_SIZE 8
#define ENTRUSTASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE

#ifdef __cplusplus
extern "C" {
#endif

/// Type of a seedhash/blockhash e.t.c.
typedef struct entrustash_h256 { uint8_t b[32]; } entrustash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// entrustash_h256_t a = entrustash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define entrustash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct entrustash_light;
typedef struct entrustash_light* entrustash_light_t;
struct entrustash_full;
typedef struct entrustash_full* entrustash_full_t;
typedef int(*entrustash_callback_t)(unsigned);

typedef struct entrustash_return_value {
	entrustash_h256_t result;
	entrustash_h256_t mix_hash;
	bool success;
} entrustash_return_value_t;

/**
 * Allocate and initialize a new entrustash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated entrustash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref entrustash_compute_cache_nodes()
 */
entrustash_light_t entrustash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated entrustash_light handler
 * @param light        The light handler to free
 */
void entrustash_light_delete(entrustash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of entrustash_return_value_t holding the return values
 */
entrustash_return_value_t entrustash_light_compute(
	entrustash_light_t light,
	entrustash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new entrustash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref entrustash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated entrustash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref entrustash_compute_full_data()
 */
entrustash_full_t entrustash_full_new(entrustash_light_t light, entrustash_callback_t callback);

/**
 * Frees a previously allocated entrustash_full handler
 * @param full    The light handler to free
 */
void entrustash_full_delete(entrustash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of entrustash_return_value to hold the return value
 */
entrustash_return_value_t entrustash_full_compute(
	entrustash_full_t full,
	entrustash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* entrustash_full_dag(entrustash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t entrustash_full_dag_size(entrustash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
entrustash_h256_t entrustash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
