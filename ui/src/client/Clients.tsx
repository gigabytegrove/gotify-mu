import React, {useEffect, useState} from 'react';
import {
    Button,
    Chip,
    IconButton,
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    Tooltip,
} from '@mui/material';
import Delete from '@mui/icons-material/Delete';
import Edit from '@mui/icons-material/Edit';
import Security from '@mui/icons-material/Security';
import ConfirmDialog from '../common/ConfirmDialog';
import DefaultPage from '../common/DefaultPage';
import SurfaceCard from '../common/SurfaceCard';
import AddClientDialog from './AddClientDialog';
import UpdateClientDialog from './UpdateClientDialog';
import ElevateClientDialog from './ElevateClientDialog';
import {IClient} from '../types';
import {LastUsedCell} from '../common/LastUsedCell';
import {formatDate} from '../common/TimeAgoFormatter';
import {RemainingTime} from '../common/RemainingTime';
import {observer} from 'mobx-react-lite';
import {useStores} from '../stores';
import {TokenConfirmDialog} from '../common/TokenConfirmDialog';

const Clients = observer(() => {
    const {clientStore, currentUser} = useStores();
    const [toDeleteClient, setToDeleteClient] = useState<IClient>();
    const [toUpdateClient, setToUpdateClient] = useState<IClient>();
    const [toElevateClient, setToElevateClient] = useState<IClient>();
    const [createDialog, setCreateDialog] = useState(false);
    const [toShowToken, setToShowToken] = useState('');
    const clients = clientStore.getItems();

    useEffect(() => void clientStore.refresh(), []);

    return (
        <DefaultPage
            title="Clients"
            description="Manage browser, mobile, and API client credentials for your account."
            rightControl={
                <Button
                    id="create-client"
                    variant="contained"
                    onClick={() => setCreateDialog(true)}>
                    Create Client
                </Button>
            }>
            <SurfaceCard
                title="Authorized Clients"
                subtitle={`${clients.length} client${clients.length === 1 ? '' : 's'}`}>
                <Table id="client-table">
                    <TableHead>
                        <TableRow>
                            <TableCell>Name</TableCell>
                            <TableCell>Status</TableCell>
                            <TableCell>Elevation</TableCell>
                            <TableCell>Expires</TableCell>
                            <TableCell>Last Used</TableCell>
                            <TableCell>Created</TableCell>
                            <TableCell align="right">Actions</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {clients.map((client: IClient) => (
                            <Row
                                key={client.id}
                                name={client.name}
                                createdAt={client.createdAt}
                                lastUsed={client.lastUsed}
                                elevatedUntil={client.elevatedUntil}
                                expiresAt={client.expiresAt}
                                current={client.id === currentUser.user.clientId}
                                fEdit={() => setToUpdateClient(client)}
                                fDelete={() => setToDeleteClient(client)}
                                fElevate={() => setToElevateClient(client)}
                            />
                        ))}
                    </TableBody>
                </Table>
            </SurfaceCard>

            {toShowToken && (
                <TokenConfirmDialog token={toShowToken} fClose={() => setToShowToken('')} />
            )}
            {createDialog && (
                <AddClientDialog
                    fClose={(token) => {
                        setCreateDialog(false);
                        setToShowToken(token ?? '');
                    }}
                    fOnSubmit={clientStore.create}
                />
            )}
            {toUpdateClient && (
                <UpdateClientDialog
                    fClose={() => setToUpdateClient(undefined)}
                    fOnSubmit={(name, expiresAfterInactivitySeconds) =>
                        clientStore.update(toUpdateClient.id, name, expiresAfterInactivitySeconds)
                    }
                    initialName={toUpdateClient.name}
                    initialExpiresAfterInactivitySeconds={
                        toUpdateClient.expiresAfterInactivitySeconds
                    }
                />
            )}
            {toDeleteClient && (
                <ConfirmDialog
                    title="Delete Client"
                    text={`Delete ${toDeleteClient.name}? Its token will stop working immediately.`}
                    fClose={() => setToDeleteClient(undefined)}
                    fOnSubmit={() => clientStore.remove(toDeleteClient.id)}
                    requireElevated
                />
            )}
            {toElevateClient && (
                <ElevateClientDialog
                    clientName={toElevateClient.name}
                    clientId={toElevateClient.id}
                    fClose={() => setToElevateClient(undefined)}
                />
            )}
        </DefaultPage>
    );
});

interface IRowProps {
    name: string;
    createdAt: string;
    lastUsed: string | null;
    elevatedUntil?: string;
    expiresAt: string | null;
    current: boolean;
    fEdit: VoidFunction;
    fDelete: VoidFunction;
    fElevate: VoidFunction;
}

const Row = ({
    name,
    createdAt,
    lastUsed,
    elevatedUntil,
    expiresAt,
    current,
    fEdit,
    fDelete,
    fElevate,
}: IRowProps) => (
    <TableRow hover selected={current} aria-current={current ? 'true' : undefined}>
        <TableCell>
            <strong className="name">{name}</strong>
        </TableCell>
        <TableCell>
            <Chip size="small" label={current ? 'Current' : 'Authorized'} variant="outlined" />
        </TableCell>
        <TableCell title={elevatedUntil}>
            <RemainingTime
                until={
                    elevatedUntil && Date.parse(elevatedUntil) > Date.now()
                        ? elevatedUntil
                        : undefined
                }
            />
        </TableCell>
        <TableCell className="expires-in" title={expiresAt ?? undefined}>
            <RemainingTime until={expiresAt} />
        </TableCell>
        <TableCell>
            <LastUsedCell lastUsed={lastUsed} />
        </TableCell>
        <TableCell title={createdAt}>{formatDate(createdAt)}</TableCell>
        <TableCell align="right">
            <Tooltip title="Elevate client">
                <IconButton onClick={fElevate} className="elevate">
                    <Security />
                </IconButton>
            </Tooltip>
            <Tooltip title="Edit client">
                <IconButton onClick={fEdit} className="edit">
                    <Edit />
                </IconButton>
            </Tooltip>
            <Tooltip title="Delete client">
                <IconButton onClick={fDelete} className="delete">
                    <Delete />
                </IconButton>
            </Tooltip>
        </TableCell>
    </TableRow>
);

export default Clients;
