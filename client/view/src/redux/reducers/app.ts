import { fromJS } from 'immutable';
import actions, { ActionsType } from '../actions';
import { getType } from 'typesafe-actions';
import { message } from 'antd'

export const initialState = fromJS({
  loading: false,
  error: false,
  user: undefined
});

export function reducer(state = initialState, action: ActionsType) {
  const result: any = action.payload;
  switch (action.type) {
    case getType(actions.resLogin):
      if (result.code === 0) {
        return state.set('user', result.data);
      }
      return state;
    case getType(actions.resLoginCheck):
      if (result.code === 0) {
        return state.set('user', result.data);
      } else {
        if (result.message) {
          const msg = result.message;
          result.message = '';
          return state.set('loginCheckMessage', msg)
        }
      }
      return state;
    case getType(actions.resRegister):
      if (result.code === 0) {
        return state.set('user', result.data && result.data.user || null);
      }
    return state
    default:
      return state;
  }
}
