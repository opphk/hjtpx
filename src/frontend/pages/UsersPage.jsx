import React, { useState, useEffect } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useApi } from '../hooks/useApi';
import Table from '../components/Table';
import Input from '../components/Input';
import Button from '../components/Button';
import Alert from '../components/Alert';
import Loading from '../components/Loading';
import '../styles/components.css';

function UsersPage() {
  const { user: currentUser } = useAuth();
  const { get } = useApi();
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [searchTerm, setSearchTerm] = useState('');
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const itemsPerPage = 10;

  useEffect(() => {
    fetchUsers();
  }, [currentPage, searchTerm]);

  const fetchUsers = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await get('/api/users', {
        page: currentPage,
        limit: itemsPerPage,
        search: searchTerm
      });
      setUsers(response.data || []);
      setTotalPages(response.totalPages || 1);
    } catch (err) {
      setError(err.message || 'Failed to fetch users');
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e) => {
    e.preventDefault();
    setCurrentPage(1);
    fetchUsers();
  };

  const handlePageChange = (page) => {
    if (page >= 1 && page <= totalPages) {
      setCurrentPage(page);
    }
  };

  const columns = [
    { key: 'id', label: 'ID' },
    { key: 'username', label: 'Username' },
    { key: 'email', label: 'Email' },
    { key: 'role', label: 'Role' },
    { key: 'created_at', label: 'Created At' }
  ];

  const filteredUsers = users.filter(user =>
    user.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.email.toLowerCase().includes(searchTerm.toLowerCase())
  );

  if (loading && users.length === 0) {
    return <Loading fullScreen />;
  }

  return (
    <div className="users-page">
      <header className="users-header">
        <h1>Users Management</h1>
        {currentUser && <span>Welcome, {currentUser.username}</span>}
      </header>

      {error && <Alert type="error" message={error} onClose={() => setError('')} />}

      <div className="users-controls">
        <form onSubmit={handleSearch} className="search-form">
          <Input
            type="text"
            name="search"
            placeholder="Search users..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
          <Button type="submit">Search</Button>
        </form>
        <Button onClick={fetchUsers}>Refresh</Button>
      </div>

      <div className="users-table-container">
        {loading ? (
          <Loading />
        ) : (
          <Table
            data={filteredUsers}
            columns={columns}
            currentPage={currentPage}
            totalPages={totalPages}
            onPageChange={handlePageChange}
          />
        )}
      </div>

      {filteredUsers.length === 0 && !loading && (
        <div className="no-users">
          <p>No users found.</p>
        </div>
      )}
    </div>
  );
}

export default UsersPage;
