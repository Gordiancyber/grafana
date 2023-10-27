import { debounce } from 'lodash';

import { getBackendSrv } from '@grafana/runtime';
import { FetchDataArgs } from '@grafana/ui';
import { accessControlQueryParam } from 'app/core/utils/accessControl';
import { OrgUser } from 'app/types';

import { ThunkResult } from '../../../types';

import {
  usersLoaded,
  pageChanged,
  usersFetchBegin,
  usersFetchEnd,
  searchQueryChanged,
  sortChanged,
  usersRolesLoaded,
} from './reducers';

export function loadUsers(withRoles = false, orgId?: number): ThunkResult<void> {
  return async (dispatch, getState) => {
    try {
      const { perPage, page, searchQuery, sort } = getState().users;
      const users = await getBackendSrv().get(
        `/api/org/users/search`,
        accessControlQueryParam({ perpage: perPage, page, query: searchQuery, sort })
      );
      dispatch(usersLoaded(users));

      if (withRoles) {
        const userIds = users?.orgUsers.map((u: OrgUser) => u.userId);
        dispatch(loadUsersRoles(userIds, orgId));
      }
    } catch (error) {
      usersFetchEnd();
    }
  };
}

export function loadUsersRoles(userIds: number[], orgId?: number): ThunkResult<void> {
  return async (dispatch, getState) => {
    try {
      const roles = await getBackendSrv().get(
        `/api/access-control/users/roles`,
        accessControlQueryParam({ userIds, targetOrgId: orgId })
      );
      dispatch(usersRolesLoaded(roles));
    } catch (error) {
      console.log(error);
    }
  };
}

const fetchUsersWithDebounce = debounce((dispatch) => dispatch(loadUsers()), 300);

export function updateUser(user: OrgUser): ThunkResult<void> {
  return async (dispatch) => {
    await getBackendSrv().patch(`/api/org/users/${user.userId}`, { role: user.role });
    dispatch(loadUsers());
  };
}

export function removeUser(userId: number): ThunkResult<void> {
  return async (dispatch) => {
    await getBackendSrv().delete(`/api/org/users/${userId}`);
    dispatch(loadUsers());
  };
}

export function changePage(page: number): ThunkResult<void> {
  return async (dispatch) => {
    dispatch(usersFetchBegin());
    dispatch(pageChanged(page));
    dispatch(loadUsers());
  };
}

export function changeSort({ sortBy }: FetchDataArgs<OrgUser>): ThunkResult<void> {
  const sort = sortBy.length ? `${sortBy[0].id}-${sortBy[0].desc ? 'desc' : 'asc'}` : undefined;
  return async (dispatch) => {
    dispatch(usersFetchBegin());
    dispatch(sortChanged(sort));
    dispatch(loadUsers());
  };
}

export function changeSearchQuery(query: string): ThunkResult<void> {
  return async (dispatch) => {
    dispatch(usersFetchBegin());
    dispatch(searchQueryChanged(query));
    fetchUsersWithDebounce(dispatch);
  };
}
