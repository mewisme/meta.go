import assert from "node:assert/strict"
import test from "node:test"
import {ClientError, Status} from "nice-grpc"
import {MetaRpcError, mapRpcError} from "../src/index.js"

test("mapRpcError converts nice-grpc errors", () => {
  const source = new ClientError("/meta.v1.RuntimeService/GetInfo", Status.UNAUTHENTICATED, "missing bearer token")
  const error = mapRpcError(source)
  assert.ok(error instanceof MetaRpcError)
  assert.equal(error.code, Status.UNAUTHENTICATED)
  assert.equal(error.path, source.path)
  assert.equal(error.cause, source)
})
