# TTL (Time To Live) Cache Implementation for Go

[![Build & Test Action Status](https://github.com/devnw/ttl/actions/workflows/build.yml/badge.svg)](https://github.com/devnw/ttl/actions)
[![Go Report Card](https://goreportcard.com/badge/go.devnw.com/ttl)](https://goreportcard.com/report/go.devnw.com/ttl)
[![codecov](https://codecov.io/gh/devnw/ttl/branch/main/graph/badge.svg)](https://codecov.io/gh/devnw/ttl)
[![Go Reference](https://pkg.go.dev/badge/go.devnw.com/ttl.svg)](https://pkg.go.dev/go.devnw.com/ttl)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](http://makeapullrequest.com)

## Overview

A TTL (Time To Live) cache is a data store where data is removed after a defined period of time to optimize memory performance.

### Implementation

#### TTL Details

This TTL Cache implementation is designed for ease of use. There are two methods for setting TTL for a data element.

1. TTL is configured when the Cache is instantiated using `ttl.NewCache` which accepts a `time.Duration` parameter for the default TTL of the cache instance. Unless otherwise specified for a specific data element, this is the TTL for all data stored in the cache.

2. TTL for each data element can be configured using the `cache.SetTTL` meethod which accepts a `time.Duration` which will set a specific TTL for that piece of data when it's loaded into the cache.

#### TTL Extension

In order to optimize data access and retention an `extension` option was included as part of the implementation of the TTL cache. This can be enabled as part of the `NewCache` method. TTL Extension configures the cache so that when data is `READ` from the cache the timeout for that **key/value** pair is reset extending the life of that data in memory. This allows for regularly accessed data to be retained while less regularly accessed data is cleared from memory.

## Usage

`go get -u go.devnw.com/ttl@latest`

### Cache Creation

```go
cache := ttl.NewCache(
    ctx, // Context used in the application
    timeout, // `time.Duration` that configures the default timeout for elements of the cache
    extend, // boolean which configures whether or not the timeout should be reset on READ
)
```

The `NewCache` method returns the `ttl.Cache` interface which defines the expected API for storing and accessing data in the cache. 

### Add Data to Cache

Adding to the cache uses a simple key/value pattern for setting data. There are two functions for adding data to the cache. The standard method `Set` uses the cache's configured default timeout.

`err := cache.Set(ctx, key, value)`

The `SetTTL` method uses the timeout (`time.Duration`) that is passed into the method for configuring how long the cache should hold the data before it's cleared from memory.

`err := cache.SetTTL(ctx, key, value, timeout)`

### Get Data from Cache

Getting data from the cache follows a fairly standard pattern which is similar ot the `sync.Map` get method.

`value, exists := cache.Get(ctx, key)`

The Get method returns the value (if it exists) and a boolean which indicates if a value was retrieved from the cache.

### Delete Data from Cache

As with Get, Delete uses a similar pattern to `sync.Map`.

`cache.Delete(ctx, key)`

This deletes the key from the map as well as shuts down the backend routines running that key's processing.