import {ClientError, Status, type CallOptions, type ClientMiddleware, type ClientMiddlewareCall} from "nice-grpc"

export class MetaError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options)
    this.name = new.target.name
  }
}

export class MetaRpcError extends MetaError {
  constructor(public readonly path: string, public readonly code: Status, public readonly details: string, options?: ErrorOptions) {
    super(`${path} ${Status[code]}: ${details}`, options)
  }
}

export class UnsupportedCapabilityError extends MetaError {
  constructor(public readonly capability: string) { super(`runtime does not support capability ${capability}`) }
}

export class RuntimeLaunchError extends MetaError {}

export class ProtocolMismatchError extends MetaError {
  constructor(public readonly expected: number, public readonly actual: number) { super(`runtime protocol major ${actual} is incompatible with SDK protocol major ${expected}`) }
}

export class EventBufferOverflowError extends MetaError {
  constructor(public readonly limit: number) { super(`event iterator exceeded its local buffer limit of ${limit}`) }
}

export function mapRpcError(error: unknown): unknown {
  if (error instanceof ClientError) return new MetaRpcError(error.path, error.code, error.details, {cause: error})
  return error
}

export const errorMiddleware: ClientMiddleware = async function* <Request, Response>(call: ClientMiddlewareCall<Request, Response>, options: CallOptions) {
  try {
    return yield* call.next(call.request, options)
  } catch (error) {
    throw mapRpcError(error)
  }
}
