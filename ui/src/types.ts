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
    acknowledgementCount?: number;
    acknowledgedBy?: IMessageAcknowledgement[];
    parentMessageId?: number;
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
    allowedCidrs?: string;
    requireSignature?: boolean;
    signingSecretConfigured?: boolean;
    signingSecret?: string;
    maxAgeSeconds?: number;
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
    scheduleType: 'once' | 'hourly' | 'daily' | 'weekly' | 'cron';
    runAt?: string;
    hour: number;
    minute: number;
    weekday: number;
    cronExpression?: string;
    timezone: string;
    endAt?: string;
    maxRuns: number;
    runCount: number;
    misfirePolicy: 'send' | 'skip';
    excludeDates?: string;
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


export interface IMessageAcknowledgement {
    userId: number;
    name: string;
    displayName?: string;
    acknowledgedAt: string;
}

export interface IIntegrationStatus {
    kind: string;
    objectId: number;
    state: string;
    lastConnectedAt?: string;
    lastActivityAt?: string;
    lastError?: string;
    updatedAt: string;
}

export interface IAutomationRun {
    id: number;
    kind: string;
    objectId: number;
    triggerKey: string;
    status: string;
    messageId?: number;
    error?: string;
    startedAt: string;
    finishedAt?: string;
}

export interface ISecurityPolicy {
    id: number;
    minPasswordLength: number;
    sessionLifetimeHours: number;
    elevationMinutes: number;
    auditRetentionDays: number;
    automationRetentionDays: number;
    allowNativePluginUploads: boolean;
    requirePluginChecksum: boolean;
    requirePluginSignature: boolean;
    requireMfaAdmins: boolean;
    requireMfaAll: boolean;
    updatedAt: string;
}


export interface IMFAStatus {
    enabled: boolean;
    recoveryCodesRemaining: number;
}


export interface IRSSIntegration {
    id: number;
    name: string;
    applicationId: number;
    url: string;
    pollMinutes: number;
    titlePrefix: string;
    enabled: boolean;
    createdAt: string;
    updatedAt: string;
}

export interface ICalendarIntegration {
    id: number;
    name: string;
    applicationId: number;
    url: string;
    pollMinutes: number;
    advanceMinutes: number;
    enabled: boolean;
    createdAt: string;
    updatedAt: string;
}

export interface IEmailGateway {
    id: number;
    name: string;
    applicationId: number;
    host: string;
    port: number;
    useTls: boolean;
    startTls: boolean;
    username: string;
    passwordConfigured: boolean;
    fromAddress: string;
    toAddresses: string;
    minPriority: number;
    enabled: boolean;
    createdAt: string;
    updatedAt: string;
}

export interface ISMTPReceiver {
    id: number;
    listenAddress: string;
    username: string;
    passwordConfigured: boolean;
    allowedCidrs: string;
    maxMessageBytes: number;
    enabled: boolean;
    updatedAt: string;
}

export interface ISMTPRoute {
    id: number;
    recipient: string;
    applicationId: number;
    enabled: boolean;
    createdAt: string;
    updatedAt: string;
}

export interface ISyslogReceiver {
    id: number;
    name: string;
    applicationId: number;
    listenAddress: string;
    protocol: 'udp' | 'tcp';
    allowedCidrs: string;
    minSeverity: number;
    enabled: boolean;
    createdAt: string;
    updatedAt: string;
}

export interface IAdminSession {
    id: number;
    userId: number;
    username: string;
    displayName?: string;
    name: string;
    createdAt: string;
    lastUsed?: string;
    elevatedUntil?: string;
    expiresAt?: string;
}

export interface ISystemStats {
    users: number;
    channels: number;
    messages: number;
    clients: number;
    groups: number;
    webhooks: number;
    mqttConnections: number;
    homeAssistantConnections: number;
    schedules: number;
    escalationRules: number;
    auditEvents: number;
    automationRuns: number;
    databaseBytes?: number;
    dataBytes?: number;
}
