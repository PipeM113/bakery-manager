// A failure of the server (5xx) carries the identifier of the request, so a person can read it out and it can be
// found in the log of the backend. A refusal of the person's own input (4xx) does not need one.
import { test } from "node:test";
import assert from "node:assert/strict";
import { apiErrorMessage } from "../src/utils/apiError.ts";

const failed = (status, error, requestId) => ({ response: { status, data: error ? { error } : {}, headers: requestId ? { "x-request-id": requestId } : {} } });

test("E1 a server failure shows its message and the code of the request", () => {
  assert.equal(apiErrorMessage(failed(500, "error eliminando receta", "d0ff621e1b63e18a"), "x"), "error eliminando receta (código d0ff621e1b63e18a)");
  assert.equal(apiErrorMessage(failed(503, "", "abc123abc123abc1"), "No se pudo guardar"), "No se pudo guardar (código abc123abc123abc1)");
});

test("E2 a refusal of the input (4xx) does not carry a code", () => {
  assert.equal(apiErrorMessage(failed(409, "No se puede eliminar: tiene 1 cotización", "d0ff621e1b63e18a"), "x"), "No se puede eliminar: tiene 1 cotización");
  assert.equal(apiErrorMessage(failed(400, "", "d0ff621e1b63e18a"), "fallback"), "fallback");
});

test("E3 without the header, or with an odd one, the message is as before", () => {
  assert.equal(apiErrorMessage(failed(500, "boom"), "x"), "boom");
  assert.equal(apiErrorMessage(failed(500, "boom", "not a code!\nINJECTED"), "x"), "boom", "only a hexadecimal code is shown");
  assert.equal(apiErrorMessage(new Error("network"), "fallback"), "fallback");
  assert.equal(apiErrorMessage(undefined, "fallback"), "fallback");
});
