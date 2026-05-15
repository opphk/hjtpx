import React, { useState, useEffect } from 'react';
import Modal from './ui/Modal';
import Input from './ui/Input';
import Button from './ui/Button';
import Alert from './ui/Alert';

const UserEditModal = ({ isOpen, onClose, user, onSave }) => {
  const [formData, setFormData] = useState({
    username: '',
    email: '',
    role: ''
  });
  const [errors, setErrors] = useState({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (user) {
      setFormData({
        username: user.username || '',
        email: user.email || '',
        role: user.role || 'user'
      });
    }
  }, [user]);

  const handleChange = (e) => {
    const { name, value } = e.target;
    setFormData(prev => ({
      ...prev,
      [name]: value
    }));
    
    if (errors[name]) {
      setErrors(prev => ({
        ...prev,
        [name]: ''
      }));
    }
  };

  const validate = () => {
    const newErrors = {};
    
    if (!formData.username.trim()) {
      newErrors.username = '用户名不能为空';
    }
    
    if (!formData.email.trim()) {
      newErrors.email = '邮箱不能为空';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = '请输入有效的邮箱地址';
    }
    
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validate()) return;
    
    setLoading(true);
    try {
      await onSave({
        ...user,
        ...formData
      });
    } finally {
      setLoading(false);
    }
  };

  const footer = (
    <>
      <Button variant="secondary" onClick={onClose}>
        取消
      </Button>
      <Button 
        variant="primary" 
        onClick={handleSubmit}
        loading={loading}
        aria-label={loading ? '保存用户信息，请等待' : '保存用户信息'}
      >
        保存
      </Button>
    </>
  );

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="编辑用户"
      footer={footer}
      size="medium"
    >
      <div className="user-edit-form">
        <Input
          label="用户名"
          name="username"
          value={formData.username}
          onChange={handleChange}
          error={errors.username}
          required
        />
        
        <Input
          label="邮箱"
          name="email"
          type="email"
          value={formData.email}
          onChange={handleChange}
          error={errors.email}
          required
        />
        
        <div className="form-group">
          <label htmlFor="role" className="form-label">
            角色<span className="required">*</span>
          </label>
          <select
            id="role"
            name="role"
            value={formData.role}
            onChange={handleChange}
            className="form-select"
            required
            aria-required="true"
            aria-describedby="role-description"
          >
            <option value="user">普通用户</option>
            <option value="admin">管理员</option>
          </select>
          <span id="role-description" className="sr-only">
            选择用户的角色权限，普通用户或管理员
          </span>
        </div>
      </div>
    </Modal>
  );
};

export default UserEditModal;
