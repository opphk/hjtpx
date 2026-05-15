import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import Table from './ui/Table';
import Button from './ui/Button';
import Loading from './ui/Loading';
import Pagination from './ui/Pagination';
import Alert from './ui/Alert';
import UserEditModal from './UserEditModal';

const UserList = () => {
  const { t } = useTranslation();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalUsers, setTotalUsers] = useState(0);
  const [editingUser, setEditingUser] = useState(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const pageSize = 10;

  useEffect(() => {
    fetchUsers();
  }, [currentPage]);

  const fetchUsers = async () => {
    setLoading(true);
    setError('');
    
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch(`/api/users?page=${currentPage}&limit=${pageSize}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        setUsers(data.users || []);
        setTotalUsers(data.total || 0);
      } else {
        setError(t('users.fetchFailed'));
      }
    } catch (err) {
      setError(t('users.networkError'));
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (user) => {
    setEditingUser(user);
    setIsModalOpen(true);
  };

  const handleDelete = async (userId) => {
    if (!window.confirm(t('users.deleteConfirm'))) return;
    
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch(`/api/users/${userId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      
      if (response.ok) {
        Alert.success(t('users.deleteSuccess'));
        fetchUsers();
      } else {
        setError(t('users.deleteFailed') || t('users.fetchFailed'));
      }
    } catch (err) {
      setError(t('users.networkError'));
    }
  };

  const handleSaveUser = async (updatedUser) => {
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch(`/api/users/${updatedUser.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(updatedUser)
      });
      
      if (response.ok) {
        setIsModalOpen(false);
        setEditingUser(null);
        fetchUsers();
      } else {
        setError(t('users.updateFailed') || t('users.fetchFailed'));
      }
    } catch (err) {
      setError(t('users.networkError'));
    }
  };

  const columns = [
    {
      title: t('table.id'),
      dataIndex: 'id',
      width: '10%'
    },
    {
      title: t('table.username'),
      dataIndex: 'username',
      width: '20%'
    },
    {
      title: t('table.email'),
      dataIndex: 'email',
      width: '25%'
    },
    {
      title: t('table.role'),
      dataIndex: 'role',
      width: '15%',
      render: (role) => (
        <span className={`badge badge-${role}`}>
          {role === 'admin' ? t('users.admin') : t('users.user')}
        </span>
      )
    },
    {
      title: t('table.actions'),
      width: '30%',
      render: (_, record) => (
        <div className="action-buttons">
          <Button 
            size="small" 
            variant="primary"
            onClick={() => handleEdit(record)}
          >
            {t('common.edit')}
          </Button>
          <Button 
            size="small" 
            variant="danger"
            onClick={() => handleDelete(record.id)}
          >
            {t('common.delete')}
          </Button>
        </div>
      )
    }
  ];

  if (loading && users.length === 0) {
    return <Loading text={t('users.loadingUsers')} />;
  }

  return (
    <div className="user-list-container">
      {error && (
        <Alert 
          type="error" 
          message={error} 
          closable 
          onClose={() => setError('')}
        />
      )}
      
      <div className="user-list-header">
        <h2>{t('users.title')}</h2>
      </div>
      
      <Table 
        columns={columns}
        data={users}
        loading={loading}
        onRowClick={(record) => handleEdit(record)}
      />
      
      {!loading && users.length > 0 && (
        <Pagination
          current={currentPage}
          total={totalUsers}
          pageSize={pageSize}
          onChange={setCurrentPage}
        />
      )}
      
      <UserEditModal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingUser(null);
        }}
        user={editingUser}
        onSave={handleSaveUser}
      />
    </div>
  );
};

export default UserList;
