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
    acknowledgementCount?: number;
    lastAcknowledgedBy?: string;
    lastAcknowledgedAt?: string;
    parentMessageId?: number;
    rootMessageId?: number;
    escalationRuleId?: number;
    escalationDepth?: number;
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
    mfaEnabled?: boolean;
    mfaRequired?: boolean;
    authProvider?: 'local' | 'oidc' | 'ldap' | 'passkey';
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
    requireSignature: boolean;
    allowedCidrs: string;
    rateLimitPerMinute: number;
    path: string;
    titleField: string;
    messageField: string;
    priorityField: string;
    defaultTitle: string;
    defaultPriority: number;
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
    status: string;
    lastConnectedAt?: string;
    lastMessageAt?: string;
    lastError?: string;
    lastErrorAt?: string;
    reconnectCount: number;
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
    status: string;
    lastConnectedAt?: string;
    lastEventAt?: string;
    lastError?: string;
    lastErrorAt?: string;
    reconnectCount: number;
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
    excludedDates?: string;
    timezone: string;
    endAt?: string;
    maxRuns: number;
    runCount: number;
    misfirePolicy: 'send' | 'skip';
    enabled: boolean;
    lastRunAt?: string;
    nextRunAt?: string;
    lastStatus?: string;
    lastError?: string;
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
    mode: 'suppress' | 'defer';
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
    targetType: 'channel' | 'user' | 'group';
    targetId: number;
    minPriority: number;
    delayMinutes: number;
    repeatMinutes: number;
    maxRepeats: number;
    enabled: boolean;
    createdAt: string;
    updatedAt: string;
}


export interface IMessageAcknowledgement {
    userId: number;
    username: string;
    displayName?: string;
    acknowledgedAt: string;
}


export interface IScheduledNotificationRun {
    id: number;
    scheduleId: number;
    scheduledFor: string;
    startedAt: string;
    finishedAt?: string;
    status: string;
    messageId?: number;
    error?: string;
}


export interface IMFAStatus {
    enabled: boolean;
    enrolledAt?: string;
    recoveryCodes: number;
}

export interface IMFASetupResult {
    secret: string;
    provisioningUri: string;
    recoveryCodes: string[];
}

export interface ISecurityPolicy {
    minimumPasswordLength: number;
    sessionInactivityMinutes: number;
    elevationMinutes: number;
    requireMfaForAdmins: boolean;
    requireMfaForAllLocalUsers: boolean;
    auditRetentionDays: number;
}

export interface IOperationsSummary {
    users: number;
    channels: number;
    messages: number;
    clients: number;
    plugins: number;
    webhooks: number;
    mqttConnections: number;
    homeAssistantConnections: number;
    schedules: number;
    pendingEscalations: number;
    pendingDigests: number;
    deferredNotifications: number;
    auditEvents: number;
    databaseDialect: string;
}

export interface IAdminSession {
    id: number;
    userId: number;
    username: string;
    name: string;
    createdAt: string;
    lastUsed?: string;
    elevatedUntil?: string;
    expiresAt?: string;
}


export interface IEmailGateway {
    id: number;
    name: string;
    sourceApplicationId: number;
    smtpHost: string;
    smtpPort: number;
    tlsMode: 'none' | 'starttls' | 'tls';
    username: string;
    passwordConfigured: boolean;
    fromAddress: string;
    toAddresses: string;
    minPriority: number;
    enabled: boolean;
    status: string;
    lastSentAt?: string;
    lastError?: string;
    lastErrorAt?: string;
}

export interface ISMTPRoute {
    id: number;
    name: string;
    applicationId: number;
    recipient: string;
    allowedCidrs: string;
    username: string;
    passwordConfigured: boolean;
    enabled: boolean;
}

export interface IRSSMonitor {
    id: number;
    name: string;
    applicationId: number;
    url: string;
    intervalMinutes: number;
    enabled: boolean;
    status: string;
    lastCheckedAt?: string;
    lastItemAt?: string;
    lastError?: string;
    lastErrorAt?: string;
}

export interface ISyslogRoute {
    id: number;
    name: string;
    applicationId: number;
    facility: number;
    maxSeverity: number;
    allowedCidrs: string;
    enabled: boolean;
}

export interface ICalendarMonitor {
    id: number;
    name: string;
    applicationId: number;
    url: string;
    intervalMinutes: number;
    notifyBeforeMinutes: number;
    enabled: boolean;
    status: string;
    lastCheckedAt?: string;
    lastEventAt?: string;
    lastError?: string;
    lastErrorAt?: string;
}


export interface IPluginCatalogEntry {
    name: string;
    modulePath: string;
    version: string;
    description: string;
    website?: string;
    downloadUrl: string;
    sha256: string;
    signature: string;
    publicKey: string;
    installed: boolean;
}


export interface IPasskey {
    id: number;
    name: string;
    createdAt: string;
    lastUsedAt?: string;
}
