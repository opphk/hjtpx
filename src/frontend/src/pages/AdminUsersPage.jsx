import React, { useState, useEffect } from 'react';
import SEO, { getSEOMeta } from '../components/SEO';
import AdminUserTable from '../components/AdminUserTable';
import AdminUserEdit from '../components/AdminUserEdit';
import Loading from '../components/ui/Loading';
import Alert from '../components/ui/Alert';
import Button from '../components/ui/Button';
import Pagination from '../components/ui/Pagination';

const AdminUsersPage = () => {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalUsers, setTotalUsers] = useState(0);
  const [editingUser, setEditingUser] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [filters, setFilters] = useState({
    role: '',
    status: '',
    search: ''
  });
  const [selectedUsers, setSelectedUsers] = useState([]);
  const [selectAll, setSelectAll] = useState(false);
  const [bulkAction, setBulkAction] = useState('');
  const seoMeta = getSEOMeta('adminUsers');

  const pageSize = 10;

  useEffect(() => {
    fetchUsers();
  }, [currentPage, filters]);

  const fetchUsers = async () => {
    setLoading(true);
    setError('');

    try {
      const token = localStorage.getItem('authToken');
      const queryParams = new URLSearchParams({
        page: currentPage,
        limit: pageSize,
        ...filters
      });

      const response = await fetch(`/api/v1/admin/users?${queryParams}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        const data = await response.json();
        setUsers(data.users || []);
        setTotalUsers(data.total || 0);
      } else {
        const errorData = await response.json();
        setError(errorData.error || '获取用户列表失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingUser(null);
    setIsModalOpen(true);
  };

  const handleEdit = (user) => {
    setEditingUser(user);
    setIsModalOpen(true);
  };

  const handleDelete = async (userId) => {
    if (!window.confirm('确定要删除此用户吗？此操作不可撤销。')) return;

    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch(`/api/v1/admin/users/${userId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (response.ok) {
        setSuccess('用户删除成功');
        fetchUsers();
      } else {
        const errorData = await response.json();
        setError(errorData.error || '删除用户失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    }
  };

  const handleStatusChange = async (userId, newStatus) => {
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch(`/api/v1/admin/users/${userId}/status`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ status: newStatus })
      });

      if (response.ok) {
        setSuccess(`用户状态已更新为 ${newStatus === 'active' ? '激活' : '禁用'}`);
        fetchUsers();
      } else {
        const errorData = await response.json();
        setError(errorData.error || '更新用户状态失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    }
  };

  const handleRoleChange = async (userId, newRole) => {
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch(`/api/v1/admin/users/${userId}/role`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify({ role: newRole })
      });

      if (response.ok) {
        setSuccess('用户角色已更新');
        fetchUsers();
      } else {
        const errorData = await response.json();
        setError(errorData.error || '更新用户角色失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    }
  };

  const handleSaveUser = async (userData) => {
    try {
      const token = localStorage.getItem('authToken');
      const isEdit = !!editingUser;
      const url = isEdit
        ? `/api/v1/admin/users/${editingUser.id}`
        : '/api/v1/admin/users';
      const method = isEdit ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(userData)
      });

      if (response.ok) {
        setIsModalOpen(false);
        setEditingUser(null);
        setSuccess(isEdit ? '用户更新成功' : '用户创建成功');
        fetchUsers();
      } else {
        const errorData = await response.json();
        setError(errorData.error || (isEdit ? '更新用户失败' : '创建用户失败'));
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    }
  };

  const handleFilterChange = (key, value) => {
    setFilters(prev => ({ ...prev, [key]: value }));
    setCurrentPage(1);
  };

  const handleSearch = (searchTerm) => {
    handleFilterChange('search', searchTerm);
  };

  const handleSelectAll = (checked) => {
    setSelectAll(checked);
    if (checked) {
      setSelectedUsers(users.map(u => u.id));
    } else {
      setSelectedUsers([]);
    }
  };

  const handleSelectUser = (userId, checked) => {
    if (checked) {
      setSelectedUsers(prev => [...prev, userId]);
    } else {
      setSelectedUsers(prev => prev.filter(id => id !== userId));
    }
  };

  const handleBulkAction = async () => {
    if (!bulkAction) {
      setError('请选择批量操作');
      return;
    }

    if (selectedUsers.length === 0) {
      setError('请选择要操作的用户');
      return;
    }

    if (bulkAction === 'delete') {
      if (!window.confirm(`确定要删除选中的 ${selectedUsers.length} 个用户吗？此操作不可撤销。`)) {
        return;
      }
    }

    try {
      const token = localStorage.getItem('authToken');
      let successCount = 0;
      let failCount = 0;

      for (const userId of selectedUsers) {
        if (bulkAction === 'status') {
          const response = await fetch(`/api/v1/admin/users/${userId}/status`, {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({ status: 'inactive' })
          });
          if (response.ok) successCount++;
          else failCount++;
        } else if (bulkAction === 'delete') {
          const response = await fetch(`/api/v1/admin/users/${userId}`, {
            method: 'DELETE',
            headers: {
              'Authorization': `Bearer ${token}`
            }
          });
          if (response.ok) successCount++;
          else failCount++;
        }
      }

      setSuccess(`批量操作完成：成功 ${successCount} 个，失败 ${failCount} 个`);
      setSelectedUsers([]);
      setSelectAll(false);
      setBulkAction('');
      fetchUsers();
    } catch (err) {
      setError('批量操作失败，请稍后重试');
    }
  };

  if (loading && users.length === 0) {
    return <Loading text="加载用户列表..." />;
  }

  return (
    <>
      <SEO {...seoMeta} />
      <div className="admin-users-page">
      <div className="page-header">
        <div>
          <h1>用户管理</h1>
          <p>管理系统中的所有用户账户、权限和状态</p>
        </div>
        <Button variant="primary" onClick={handleCreate}>
          创建用户
        </Button>
      </div>

      {error && (
        <Alert
          type="error"
          message={error}
          closable
          onClose={() => setError('')}
        />
      )}

      {success && (
        <Alert
          type="success"
          message={success}
          closable
          onClose={() => setSuccess('')}
        />
      )}

      <div className="filter-bar">
        <div className="filter-item">
          <input
            type="text"
            placeholder="搜索用户名或邮箱..."
            value={filters.search}
            onChange={(e) => handleSearch(e.target.value)}
            className="search-input"
          />
        </div>
        <div className="filter-item">
          <select
            value={filters.role}
            onChange={(e) => handleFilterChange('role', e.target.value)}
            className="form-select"
          >
            <option value="">所有角色</option>
            <option value="admin">管理员</option>
            <option value="user">普通用户</option>
            <option value="guest">访客</option>
          </select>
        </div>
        <div className="filter-item">
          <select
            value={filters.status}
            onChange={(e) => handleFilterChange('status', e.target.value)}
            className="form-select"
          >
            <option value="">所有状态</option>
            <option value="active">激活</option>
            <option value="inactive">禁用</option>
            <option value="pending">待验证</option>
          </select>
        </div>
      </div>

      {selectedUsers.length > 0 && (
        <div className="bulk-actions">
          <span className="selected-count">已选择 {selectedUsers.length} 个用户</span>
          <select
            value={bulkAction}
            onChange={(e) => setBulkAction(e.target.value)}
            className="form-select bulk-action-select"
          >
            <option value="">选择批量操作</option>
            <option value="status">批量禁用</option>
            <option value="delete">批量删除</option>
          </select>
          <Button
            variant="primary"
            size="small"
            onClick={handleBulkAction}
          >
            应用
          </Button>
          <Button
            variant="secondary"
            size="small"
            onClick={() => {
              setSelectedUsers([]);
              setSelectAll(false);
              setBulkAction('');
            }}
          >
            取消选择
          </Button>
        </div>
      )}

      <AdminUserTable
        users={users}
        loading={loading}
        onEdit={handleEdit}
        onDelete={handleDelete}
        onStatusChange={handleStatusChange}
        onRoleChange={handleRoleChange}
        selectedUsers={selectedUsers}
        onSelectUser={handleSelectUser}
        onSelectAll={handleSelectAll}
        selectAll={selectAll}
      />

      {!loading && users.length > 0 && (
        <Pagination
          current={currentPage}
          total={totalUsers}
          pageSize={pageSize}
          onChange={setCurrentPage}
        />
      )}

      <AdminUserEdit
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingUser(null);
        }}
        user={editingUser}
        onSave={handleSaveUser}
      />
    </div>
    </>
  );
};

export default AdminUsersPage;
