export interface IApplication {
    id: number;
    token: string;
    ownerId?: number;
    autoAssign?: boolean;
    allowMemberPost?: boolean;
    channelType?: "notification" | "chat";
    receiveNotifications?: boolean;
    name: string;
    sortKey: string;
    description: string;
    image: string;
    internal: boolean;
    defaultPriority: number;
    lastUsed: string | null;
    createdAt: string;
}

export interface IClient {
    id: number;
    token: string;
    name: string;
    lastUsed: string | null;
    elevatedUntil?: string;
    createdAt: string;
    expiresAfterInactivitySeconds: number;
    expiresAt: string | null;
}

export interface IPlugin {
    id: number;
    token: string;
    name: string;
    modulePath: string;
    enabled: boolean;
    author?: string;
    website?: string;
    license?: string;
    capabilities: Array<'webhooker' | 'displayer' | 'configurer' | 'messenger' | 'storager'>;
    createdAt: string;
}

export interface IMessage {
    id: number;
    appid: number;
    message: string;
    title: string;
    priority: number;
    date: string;
    senderUserId?: number;
    senderName?: string;
    image?: string;
    extras?: IMessageExtras;
}

export interface IMessageExtras {
    [key: string]: any; // eslint-disable-line  @typescript-eslint/no-explicit-any
}

export interface IPagedMessages {
    paging: IPaging;
    messages: IMessage[];
}

export interface IPaging {
    next?: string;
    since?: number;
    size: number;
    limit: number;
}

export interface IUser {
    id: number;
    name: string;
    displayName?: string;
    admin: boolean;
    createdAt: string;
}

export interface IUserGroup {
    id: number;
    name: string;
    description: string;
    memberCount: number;
    createdAt: string;
    updatedAt: string;
}

export interface IUserGroupMember {
    userId: number;
    name: string;
    displayName?: string;
    admin: boolean;
}

export interface IAuditEvent {
    id: number;
    userId: number;
    username: string;
    action: string;
    target: string;
    targetId?: string;
    details?: string;
    ipAddress?: string;
    createdAt: string;
}

export interface ICurrentUser extends IUser {
    clientId?: number;
    elevatedUntil?: string;
}

export interface IVersion {
    version: string;
    commit: string;
    buildDate: string;
}

export interface IApplicationMember {
    userId: number;
    name: string;
    owner: boolean;
    receiveNotifications: boolean;
    autoAssigned: boolean;
}


export interface IMUTypingEvent {
    type: 'typing';
    applicationId: number;
    userId: number;
    userName: string;
    typing: boolean;
    expiresAt: string;
}

export type IMURealtimeEvent = IMUTypingEvent;
