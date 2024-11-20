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
using TInstant = i32;

using TMutex = std::mutex;

#define TVector std::vector

void Log(const TString& entity, const TString& msg) {
    std::cerr << "LOGGER: " << entity << ": " << msg << std::endl;
}

void LogClient(const TString& msg) {
    Log("CLIENT", msg);
}

void LogWorker(const TString& msg) {
    Log("WORKER", msg);
}

class TTask {
private:
    double start_ = 0;
    double end_ = 0;
    double step_ = 0;

public:
    TTask(double start, double end, double step) : start_(start), end_(end), step_(step) {}

    TTask() {}

    double GetStep() const {
        return step_;
    }

    double GetEnd() const {
        return end_;
    }

    double GetStart() const {
        return start_;
    }

    void SetStep(double step) {
        this->step_ = step;
    }

    void SetEnd(double end) {
        this->end_ = end;
    }

    void SetStart(double start) {
        this->start_ = start;
    }

    TString AsString() const {
        return "Task. start: " + std::to_string(start_) + ", end: " + std::to_string(end_) + ", step: " + std::to_string(step_);
    }
};
