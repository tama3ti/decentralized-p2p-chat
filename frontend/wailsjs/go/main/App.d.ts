// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {models} from '../models';

export function GetChatMessages(arg1:number):Promise<Array<models.Message>>;

export function GetUser(arg1:string):Promise<models.User>;

export function GetUsers():Promise<Array<models.User>>;

export function Login(arg1:string,arg2:string):Promise<models.User>;

export function Register(arg1:string,arg2:string,arg3:string):Promise<any>;

export function SendMessage(arg1:string,arg2:number):Promise<Array<models.Message>>;

export function StartUpFromJwt(arg1:string):Promise<void>;
