const { gql } = require('apollo-server');
const { DataLoader, createLoaders } = require('../../services/dataLoader');

const typeDefs = gql`
  type User {
    id: ID!
    email: String!
    name: String!
    role: String!
    posts: [Post!]!
    createdAt: String!
    updatedAt: String!
  }

  type Post {
    id: ID!
    title: String!
    content: String!
    author: User!
    comments: [Comment!]!
    tags: [String!]!
    createdAt: String!
    updatedAt: String!
  }

  type Comment {
    id: ID!
    content: String!
    author: User!
    post: Post!
    createdAt: String!
  }

  type PostConnection {
    edges: [PostEdge!]!
    pageInfo: PageInfo!
    totalCount: Int!
  }

  type PostEdge {
    node: Post!
    cursor: String!
  }

  type PageInfo {
    hasNextPage: Boolean!
    hasPreviousPage: Boolean!
    startCursor: String
    endCursor: String
  }

  type Query {
    users: [User!]!
    user(id: ID!): User
    posts(limit: Int, offset: Int): PostConnection!
    post(id: ID!): Post
    comments(postId: ID!): [Comment!]!
  }

  type Mutation {
    createUser(input: CreateUserInput!): User!
    updateUser(id: ID!, input: UpdateUserInput!): User!
    deleteUser(id: ID!): Boolean!
    
    createPost(input: CreatePostInput!): Post!
    updatePost(id: ID!, input: UpdatePostInput!): Post!
    deletePost(id: ID!): Boolean!
    
    addComment(postId: ID!, content: String!): Comment!
  }

  input CreateUserInput {
    email: String!
    name: String!
    password: String!
    role: String
  }

  input UpdateUserInput {
    email: String
    name: String
    password: String
    role: String
  }

  input CreatePostInput {
    title: String!
    content: String!
    tags: [String!]
  }

  input UpdatePostInput {
    title: String
    content: String
    tags: [String!]
  }
`;

class MockUserService {
  constructor() {
    this.users = new Map();
    this.initialize();
  }

  initialize() {
    const sampleUsers = [
      { id: '1', email: 'alice@example.com', name: 'Alice Smith', role: 'admin', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
      { id: '2', email: 'bob@example.com', name: 'Bob Johnson', role: 'user', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
      { id: '3', email: 'charlie@example.com', name: 'Charlie Brown', role: 'user', createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() }
    ];
    sampleUsers.forEach(u => this.users.set(u.id, u));
  }

  async findByIds(ids) {
    await this.simulateDelay();
    return ids.map(id => this.users.get(id) || null);
  }

  async findAll() {
    await this.simulateDelay();
    return Array.from(this.users.values());
  }

  async findById(id) {
    await this.simulateDelay();
    return this.users.get(id) || null;
  }

  async create(data) {
    await this.simulateDelay();
    const user = {
      id: String(Date.now()),
      ...data,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
    this.users.set(user.id, user);
    return user;
  }

  async update(id, data) {
    await this.simulateDelay();
    const user = this.users.get(id);
    if (!user) return null;
    const updated = { ...user, ...data, updatedAt: new Date().toISOString() };
    this.users.set(id, updated);
    return updated;
  }

  async delete(id) {
    await this.simulateDelay();
    return this.users.delete(id);
  }

  async simulateDelay() {
    return new Promise(resolve => setTimeout(resolve, Math.random() * 10));
  }
}

class MockPostService {
  constructor() {
    this.posts = new Map();
    this.comments = new Map();
    this.initialize();
  }

  initialize() {
    const samplePosts = [
      { id: '1', title: 'First Post', content: 'Content of first post', authorId: '1', tags: ['tech', 'news'], createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
      { id: '2', title: 'Second Post', content: 'Content of second post', authorId: '2', tags: ['opinion'], createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() },
      { id: '3', title: 'Third Post', content: 'Content of third post', authorId: '1', tags: ['tech'], createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() }
    ];
    samplePosts.forEach(p => this.posts.set(p.id, p));

    const sampleComments = [
      { id: '1', content: 'Great post!', postId: '1', authorId: '2', createdAt: new Date().toISOString() },
      { id: '2', content: 'Thanks for sharing', postId: '1', authorId: '3', createdAt: new Date().toISOString() }
    ];
    sampleComments.forEach(c => this.comments.set(c.id, c));
  }

  async findByIds(ids) {
    await this.simulateDelay();
    return ids.map(id => this.posts.get(id) || null);
  }

  async findByUserIds(userIds) {
    await this.simulateDelay();
    const result = {};
    userIds.forEach(userId => {
      result[userId] = Array.from(this.posts.values()).filter(p => p.authorId === userId);
    });
    return result;
  }

  async findAll(limit = 10, offset = 0) {
    await this.simulateDelay();
    const all = Array.from(this.posts.values());
    return {
      posts: all.slice(offset, offset + limit),
      total: all.length
    };
  }

  async findById(id) {
    await this.simulateDelay();
    return this.posts.get(id) || null;
  }

  async create(data) {
    await this.simulateDelay();
    const post = {
      id: String(Date.now()),
      ...data,
      tags: data.tags || [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString()
    };
    this.posts.set(post.id, post);
    return post;
  }

  async update(id, data) {
    await this.simulateDelay();
    const post = this.posts.get(id);
    if (!post) return null;
    const updated = { ...post, ...data, updatedAt: new Date().toISOString() };
    this.posts.set(id, updated);
    return updated;
  }

  async delete(id) {
    await this.simulateDelay();
    return this.posts.delete(id);
  }

  async findCommentsByPostId(postId) {
    await this.simulateDelay();
    return Array.from(this.comments.values()).filter(c => c.postId === postId);
  }

  async addComment(postId, content, authorId) {
    await this.simulateDelay();
    const comment = {
      id: String(Date.now()),
      content,
      postId,
      authorId,
      createdAt: new Date().toISOString()
    };
    this.comments.set(comment.id, comment);
    return comment;
  }

  async simulateDelay() {
    return new Promise(resolve => setTimeout(resolve, Math.random() * 10));
  }
}

const userService = new MockUserService();
const postService = new MockPostService();
const loaders = createLoaders(userService, postService);

const resolvers = {
  Query: {
    users: async () => {
      return userService.findAll();
    },

    user: async (_, { id }) => {
      return loaders.userLoader.load(id);
    },

    posts: async (_, { limit = 10, offset = 0 }) => {
      const result = await postService.findAll(limit, offset);
      return {
        edges: result.posts.map(post => ({
          node: post,
          cursor: Buffer.from(post.id).toString('base64')
        })),
        pageInfo: {
          hasNextPage: offset + limit < result.total,
          hasPreviousPage: offset > 0,
          startCursor: result.posts.length > 0 ? Buffer.from(result.posts[0].id).toString('base64') : null,
          endCursor: result.posts.length > 0 ? Buffer.from(result.posts[result.posts.length - 1].id).toString('base64') : null
        },
        totalCount: result.total
      };
    },

    post: async (_, { id }) => {
      return loaders.postLoader.load(id);
    },

    comments: async (_, { postId }) => {
      return postService.findCommentsByPostId(postId);
    }
  },

  Mutation: {
    createUser: async (_, { input }) => {
      return userService.create(input);
    },

    updateUser: async (_, { id, input }) => {
      loaders.userLoader.clear(id);
      return userService.update(id, input);
    },

    deleteUser: async (_, { id }) => {
      loaders.userLoader.clear(id);
      return userService.delete(id);
    },

    createPost: async (_, { input }, { userId }) => {
      return postService.create({ ...input, authorId: userId || '1' });
    },

    updatePost: async (_, { id, input }) => {
      loaders.postLoader.clear(id);
      return postService.update(id, input);
    },

    deletePost: async (_, { id }) => {
      loaders.postLoader.clear(id);
      return postService.delete(id);
    },

    addComment: async (_, { postId, content }, { userId }) => {
      return postService.addComment(postId, content, userId || '1');
    }
  },

  User: {
    posts: async (user) => {
      return loaders.userPostsLoader.load(user.id);
    }
  },

  Post: {
    author: async (post) => {
      return loaders.postAuthorLoader.load(post.id);
    }
  },

  Comment: {
    author: async (comment) => {
      return loaders.userLoader.load(comment.authorId);
    }
  }
};

module.exports = {
  typeDefs,
  resolvers,
  loaders,
  userService,
  postService
};
