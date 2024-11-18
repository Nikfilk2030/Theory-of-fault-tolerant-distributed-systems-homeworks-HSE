#pragma once

#include <algorithm>
#include <cstdint>
#include <cstring>
#include <future>
#include <string>
#include <iostream>
#include <thread>
#include <exception>
#include <unistd.h>
#include <vector>
#include <fcntl.h>
#include <stdio.h>
#include <queue>

#include <arpa/inet.h>

#include <sys/socket.h>

#include <netinet/in.h>

using TString = std::string;
using ui64 = std::uint64_t;
using ui32 = std::uint32_t;
using i64 = std::int64_t;
using i32 = std::int32_t;

void log(const TString& entity, const TString& msg) {
    std::cerr << "LOGGER: " << entity << ": " << msg << std::endl;
}

void log_client(const TString& msg) {
    log("CLIENT", msg);
}

void log_worker(const TString& msg) {
    log("WORKER", msg);
}

class TTask {
private:
    double start_ = 0;
    double end_ = 0;
    double step_ = 0;

public:
    TTask(double start, double end, double step) : start_(start), end_(end), step_(step) {}

    TTask() {}

    double get_step() const {
        return step_;
    }

    double get_end() const {
        return end_;
    }

    double get_start() const {
        return start_;
    }

    void set_step(double step) {
        this->step_ = step;
    }

    void set_end(double end) {
        this->end_ = end;
    }

    void set_start(double start) {
        this->start_ = start;
    }

    TString as_string() const {
        return std::to_string(start_) + " " + std::to_string(end_) + " " + std::to_string(step_);
    }
};
