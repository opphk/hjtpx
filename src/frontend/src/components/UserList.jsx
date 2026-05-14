import React, { useState, useEffect } from 'react';
import Table from '../ui/Table';
import Button from '../ui/Button';
import Loading from '../ui/Loading';
import Pagination from '../ui/Pagination';
import Alert from '../ui/Alert';
import UserEditModal from './UserEditModal';

const UserList = () => {
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
        setError('获取用户列表失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  const handleEdit = (user) => {
    setEditingUser(user);
    setIsModalOpen(true);
  };

  const handleDelete = async (userId) => {
    if (!window.confirm('确定要删除此用户吗？')) return;
    
    try {
      const token = localStorage.getItem('authToken');
      const response = await fetch(`/api/users/${userId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      
      if (response.ok) {
        Alert.success('用户删除成功');
        fetchUsers();
      } else {
        setError('删除用户失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
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
        setError('更新用户失败');
      }
    } catch (err) {
      setError('网络错误，请稍后重试');
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: '10%'
    },
    {
      title: '用户名',
      dataIndex: 'username',
      width: '20%'
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      width: '25%'
    },
    {
      title: '角色',
      dataIndex: 'role',
      width: '15%',
      render: (role) => (
        <span className={`badge badge-${role}`}>{role}</span>
      )
    },
    {
      title: '操作',
      width: '30%',
      render: (_, record) => (
        <div className="action-buttons">
          <Button 
            size="small" 
            variant="primary"
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button 
            size="small" 
            variant="danger"
            onClick={() => handleDelete(record.id)}
          >
            删除
          </Button>
        </div>
      )
    }
  ];

  if (loading && users.length === 0) {
    return <Loading text="加载用户列表..." />;
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
        <h2>用户管理</h2>
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
