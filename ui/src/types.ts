export interface IApplication {
    id: number;
    token: string;
    ownerId?: number;
    autoAssign?: boolean;
    allowMemberPost?: boolean;
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
    acknowledged?: boolean;
    acknowledgedByAnyone?: boolean;
    acknowledgedByName?: string;
    acknowledgedCount?: number;
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
    statusCode?: number;
    success?: boolean;
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


export interface IWebhookRoute {
    id: number;
    name: string;
    applicationId: number;
    enabled: boolean;
    path: string;
    titleField: string;
    messageField: string;
    priorityField: string;
    defaultTitle: string;
    defaultPriority: number;
    requireSignature: boolean;
    signingSecretConfigured: boolean;
    allowedCidrs: string;
    replayWindowSeconds: number;
    createdAt: string;
    updatedAt: string;
}

export interface IMQTTIntegration {
    id: number;
    name: string;
    applicationId: number;
    brokerUrl: string;
    clientId: string;
    username: string;
    passwordConfigured: boolean;
    topic: string;
    enabled: boolean;
    state?: string;
    statusMessage?: string;
    lastConnectedAt?: string;
    lastMessageAt?: string;
    lastErrorAt?: string;
    createdAt: string;
    updatedAt: string;
}

export interface IHomeAssistantIntegration {
    id: number;
    name: string;
    applicationId: number;
    baseUrl: string;
    tokenConfigured: boolean;
    eventType: string;
    enabled: boolean;
    state?: string;
    statusMessage?: string;
    lastConnectedAt?: string;
    lastMessageAt?: string;
    lastErrorAt?: string;
    createdAt: string;
    updatedAt: string;
}

export interface IScheduledNotification {
    id: number;
    name: string;
    applicationId: number;
    title: string;
    message: string;
    priority: number;
    scheduleType: 'once' | 'hourly' | 'daily' | 'weekly';
    runAt?: string;
    hour: number;
    minute: number;
    weekday: number;
    timezone: string;
    enabled: boolean;
    lastRunAt?: string;
    nextRunAt?: string;
    createdAt: string;
    updatedAt: string;
}

export interface IQuietHoursPolicy {
    id?: number;
    userId: number;
    enabled: boolean;
    startMinute: number;
    endMinute: number;
    timezone: string;
    allowPriority: number;
}

export interface IDigestPolicy {
    id?: number;
    userId: number;
    enabled: boolean;
    intervalMinutes: number;
    immediatePriority: number;
    lastSentAt?: string;
    nextRunAt?: string;
}

export interface IEscalationRule {
    id: number;
    name: string;
    sourceApplicationId: number;
    targetApplicationId: number;
    minPriority: number;
    delayMinutes: number;
    enabled: boolean;
    createdAt: string;
    updatedAt: string;
}
