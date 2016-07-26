import { Injectable } from '@angular/core';


export interface User {
  id: string;
  email: string;
}

export interface Context {
  canonicalURL: string;
  user: User;
}

declare var ctx: Context;


@Injectable()
export class ContextService {

  getContext() : Context {
    return ctx;
  }
}
