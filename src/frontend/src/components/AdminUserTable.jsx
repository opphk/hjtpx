import React from 'react';
import Button from './ui/Button';
import Loading from './ui/Loading';

const AdminUserTable = ({
  users,
  loading,
  onEdit,
  onDelete,
  onStatusChange,
  onRoleChange,
  selectedUsers = [],
  onSelectUser,
  onSelectAll,
  selectAll = false
}) => {
  const getStatusBadge = (status) => {
    const statusMap = {
      active: { label: '激活', className: 'status-active' },
      inactive: { label: '禁用', className: 'status-inactive' },
      pending: { label: '待验证', className: 'status-pending' }
    };
    const statusInfo = statusMap[status] || statusMap.pending;
    return <span className={`status-badge ${statusInfo.className}`}>{statusInfo.label}</span>;
  };

  const getRoleBadge = (role) => {
    const roleMap = {
      admin: { label: '管理员', className: 'role-admin' },
      user: { label: '普通用户', className: 'role-user' },
      guest: { label: '访客', className: 'role-guest' }
    };
    const roleInfo = roleMap[role] || roleMap.user;
    return <span className={`role-badge ${roleInfo.className}`}>{roleInfo.label}</span>;
  };

  if (loading && users.length === 0) {
    return <Loading text="加载中..." />;
  }

  if (!loading && users.length === 0) {
    return (
      <div className="empty-state">
        <p>暂无用户数据</p>
      </div>
    );
  }

  return (
    <div className="table-container">
      <table className="admin-table">
        <thead>
          <tr>
            <th>
              <input
                type="checkbox"
                checked={selectAll && selectedUsers.length === users.length}
                onChange={(e) => onSelectAll(e.target.checked)}
              />
            </th>
            <th>ID</th>
            <th>用户名</th>
            <th>邮箱</th>
            <th>角色</th>
            <th>状态</th>
            <th>注册时间</th>
            <th>最后登录</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {users.map((user) => (
            <tr key={user.id}>
              <td>
                <input
                  type="checkbox"
                  checked={selectedUsers.includes(user.id)}
                  onChange={(e) => onSelectUser(user.id, e.target.checked)}
                />
              </td>
              <td>{user.id}</td>
              <td>{user.username}</td>
              <td>{user.email}</td>
              <td>
                <select
                  value={user.role}
                  onChange={(e) => onRoleChange(user.id, e.target.value)}
                  className="role-select"
                  disabled={user.role === 'admin'}
                >
                  <option value="admin">管理员</option>
                  <option value="user">普通用户</option>
                  <option value="guest">访客</option>
                </select>
              </td>
              <td>
                <select
                  value={user.status}
                  onChange={(e) => onStatusChange(user.id, e.target.value)}
                  className="status-select"
                >
                  <option value="active">激活</option>
                  <option value="inactive">禁用</option>
                  <option value="pending">待验证</option>
                </select>
              </td>
              <td>{new Date(user.created_at).toLocaleDateString('zh-CN')}</td>
              <td>{user.last_login ? new Date(user.last_login).toLocaleDateString('zh-CN') : '从未登录'}</td>
              <td>
                <div className="action-buttons">
                  <Button size="small" variant="primary" onClick={() => onEdit(user)}>
                    编辑
                  </Button>
                  <Button
                    size="small"
                    variant="danger"
                    onClick={() => onDelete(user.id)}
                    disabled={user.role === 'admin'}
                  >
                    删除
                  </Button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

export default AdminUserTable;
